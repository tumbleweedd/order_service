package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cache_imp"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	internal_errors "github.com/tumbleweedd/two_services_system/order_service/internal/lib/errors"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"sync"
	"time"
)

const maxAllowedSendingTime = 5 * time.Second

type OrderService struct {
	log *slog.Logger

	orderCreator  OrderCreator
	orderGetter   OrderGetter
	orderCancaler OrderCancaler

	sendOrderEventsChan chan models.Event
	statusEventChan     chan models.Event
	done                chan struct{}

	cache cache_imp.CacheI[uuid.UUID, *models.Order]
}

type OrderCreator interface {
	Create(
		ctx context.Context,
		order *models.Order,
	) (uuid.UUID, error)
}

type OrderGetter interface {
	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) (ordersMap map[uuid.UUID]models.Order, err error)
	Status(ctx context.Context, orderUUID uuid.UUID) (int, error)
}

type OrderCancaler interface {
	Cancel(
		ctx context.Context,
		orderUUID uuid.UUID,
	) error
}

func NewOrderService(
	log *slog.Logger,

	orderCreator OrderCreator,
	orderGetter OrderGetter,
	orderCancaler OrderCancaler,

	sendOrderEventsChan chan models.Event,
	statusEventChan chan models.Event,
	done chan struct{},

	cache cache_imp.CacheI[uuid.UUID, *models.Order],
) *OrderService {
	return &OrderService{
		log:                 log,
		orderCreator:        orderCreator,
		orderGetter:         orderGetter,
		orderCancaler:       orderCancaler,
		sendOrderEventsChan: sendOrderEventsChan,
		statusEventChan:     statusEventChan,
		done:                done,
		cache:               cache,
	}
}

func (os *OrderService) Create(
	ctx context.Context,
	order *models.Order,
) (string, error) {
	const op = "services.order.Create"

	// TODO: перенести логику кеша в cache.go
	orderUUID, err := os.orderCreator.Create(ctx, order)
	if err != nil {
		os.log.Error(op, slog.String("error", err.Error()))
		return "", err
	}

	//// если заказ оплачен баллами, то отправляем событие о его создании в сервис начисления баллов
	//if order.PaymentType == models.Points {
	//	orderEvent := &models.Order{
	//		UserUUID:    order.UserUUID,
	//		OrderUUID:   orderUUID,
	//		Status:      order.Status,
	//		Products:    order.Products,
	//		WithPoints:  order.WithPoints,
	//		PaymentType: order.PaymentType,
	//	}
	//
	//	go os.sendEvent(ctx, op, os.sendOrderEventsChan, orderEvent)
	//}

	return orderUUID.String(), nil
}

// TODO: что-то тут не то. Подумать, какое тут должно быть поведеине (в частности, при закрытии каналов)
func (os *OrderService) sendEvent(ctx context.Context, op string, eventCh chan models.Event, event models.Event) {
	select {
	case <-ctx.Done():
		os.log.Warn(op, slog.String("ctx done err", ctx.Err().Error()))
		return
	case <-os.done:
		os.log.Info(op, fmt.Sprint("received the completion signal"))
		return
	case eventCh <- event:
	}
}

func (os *OrderService) Cancel(ctx context.Context, orderUUID uuid.UUID) error {
	const op = "services.order.Cancel"

	var status int
	var err error

	order, exist := os.cache.Get(orderUUID)
	if !exist {
		status, err = os.orderGetter.Status(ctx, orderUUID)
		if err != nil {
			os.log.Error(op, slog.String("get status error", err.Error()))
			return err
		}

		order.Status = models.OrderStatus(status)
	}

	switch order.Status {
	case models.OrderStatusCreated, models.OrderStatusPaid:
		if err = os.orderCancaler.Cancel(ctx, orderUUID); err != nil {
			os.log.Error(op, slog.String("error", err.Error()))
			return err
		}
	default:
		return fmt.Errorf("order cancellation error by status %v: %w", order.Status, internal_errors.ErrCancelOrderByStatus)
	}

	statusEvent := &models.StatusStruct{
		OrderUUID: orderUUID,
		Status:    models.OrderStatusCanceled,
	}

	go os.sendEvent(ctx, op, os.statusEventChan, statusEvent)

	return nil
}

func (os *OrderService) OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) ([]models.Order, error) {
	const op = "service.order.OrdersByUUIDs"

	inCacheCh := make(chan models.Order, len(UUIDs))
	notInCacheCh := make(chan uuid.UUID, len(UUIDs))

	wg := sync.WaitGroup{}

	for i := range UUIDs {
		orderUUID := UUIDs[i]

		wg.Add(1)
		go func() {
			defer wg.Done()
			value, ok := os.cache.Get(orderUUID)
			if !ok || value == nil {
				notInCacheCh <- orderUUID

				return
			}

			inCacheCh <- *value

			return
		}()
	}

	wg.Wait()
	close(inCacheCh)
	close(notInCacheCh)

	result := make([]models.Order, 0, len(UUIDs))
	for order := range inCacheCh {
		result = append(result, order)
	}

	notInCache := make([]uuid.UUID, 0, len(UUIDs))
	for orderUUID := range notInCacheCh {
		notInCache = append(notInCache, orderUUID)
	}

	os.log.InfoContext(ctx, op, slog.Int("items in cache", len(result)), slog.Int("items not in cache", len(notInCache)))

	errGroup := errgroup.Group{}
	errGroup.SetLimit(100)

	if len(notInCache) > 0 {
		ordersMap, err := os.orderGetter.OrdersByUUIDs(ctx, notInCache)
		if err != nil {
			os.log.Error(op, slog.String("get orders error", err.Error()))
			return nil, err
		}

		for orderUUID, order := range ordersMap {
			for _, product := range order.Products {
				order.TotalAmount += product.Amount
				ordersMap[orderUUID] = order
			}
		}

		for _, order := range ordersMap {
			orderCopy := order

			result = append(result, orderCopy)

			errGroup.Go(func() error {
				if evicted := os.cache.Add(orderCopy.OrderUUID, &orderCopy); evicted {
					return fmt.Errorf("cache size was exceeded")
				}

				return nil
			})
		}

		if err = errGroup.Wait(); err != nil {
			os.log.WarnContext(ctx, op, slog.String("cache size was exceeded", err.Error()))
		}

		os.log.InfoContext(ctx, op, slog.Int("orders from DB", len(ordersMap)))
	}

	return result, nil
}
