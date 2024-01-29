package services

import (
	"context"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
	"time"
)

const maxAllowedSendingTime = 5 * time.Second

type OrderService struct {
	log                 *slog.Logger
	orderCreator        OrderCreator
	orderCancaler       OrderCancaler
	sendOrderEventsChan chan models.Event
	statusEventChan     chan models.Event
}

type OrderCreator interface {
	Create(
		ctx context.Context,
		order *models.Order,
	) (uuid.UUID, error)
}

type OrderCancaler interface {
	Cancel(
		ctx context.Context,
		orderUUID string,
	) error
}

func NewOrderService(
	log *slog.Logger,
	orderCreator OrderCreator,
	orderCancaler OrderCancaler,
	sendOrderEventsChan chan models.Event,
	statusEventChan chan models.Event,
) *OrderService {
	return &OrderService{
		log:                 log,
		orderCreator:        orderCreator,
		orderCancaler:       orderCancaler,
		sendOrderEventsChan: sendOrderEventsChan,
		statusEventChan:     statusEventChan,
	}
}

func (os *OrderService) Create(
	ctx context.Context,
	order *models.Order,
) (string, error) {
	const op = "services.order.Create"

	orderUUID, err := os.orderCreator.Create(ctx, order)
	if err != nil {
		os.log.Error(op, slog.String("error", err.Error()))
		return "", err
	}

	// если заказ оплачен баллами, то отправляем событие о создании заказа в сервис начисления баллов
	if order.PaymentType == models.Points {
		orderEvent := &models.Order{
			UserUUID:    order.UserUUID,
			OrderUUID:   orderUUID.String(),
			Status:      order.Status,
			Products:    order.Products,
			WithPoints:  order.WithPoints,
			PaymentType: order.PaymentType,
		}

		go os.sendEvent(ctx, op, orderEvent)
	}

	return orderUUID.String(), nil
}

func (os *OrderService) sendEvent(ctx context.Context, op string, event models.Event) {
	select {
	case <-ctx.Done():
		close(os.sendOrderEventsChan)
		os.log.Warn(op, slog.String("ctx done err", ctx.Err().Error()))
	case os.sendOrderEventsChan <- event:
	case <-time.After(maxAllowedSendingTime):
		errMsg := "timeout sending order event to channel"
		os.log.Error(op, slog.String("error", errMsg))
		close(os.sendOrderEventsChan)
	}
}

func (os *OrderService) Cancel(ctx context.Context, orderUUID string) error {
	const op = "services.order.Cancel"

	err := os.orderCancaler.Cancel(ctx, orderUUID)
	if err != nil {
		os.log.Error(op, slog.String("error", err.Error()))
		return err
	}

	statusEvent := &models.StatusStruct{
		OrderUUID: orderUUID,
		Status:    models.OrderStatusCanceled,
	}

	go os.sendEvent(ctx, op, statusEvent)

	return nil
}
