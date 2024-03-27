package create

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cache_impl"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
)

type orderCreator interface {
	Create(ctx context.Context, order *models.Order) (uuid.UUID, error)
}

type OrderCreationService struct {
	log   *slog.Logger
	cache cache_impl.CacheI[uuid.UUID, *models.Order]

	orderCreator orderCreator
}

func New(log *slog.Logger, orderCreator orderCreator) *OrderCreationService {
	return &OrderCreationService{
		log:          log,
		orderCreator: orderCreator,
	}
}

func (os *OrderCreationService) Create(ctx context.Context, order *models.Order) (string, error) {
	const op = "services.order.Create"

	orderUUID, err := os.createOrder(ctx, order)
	if err != nil {
		return "", fmt.Errorf("%s: %v", op, err)
	}

	_ = os.cache.Add(orderUUID, order)

	os.log.InfoContext(ctx, op, fmt.Sprint("cache was updated"))

	return orderUUID.String(), nil
}

func (os *OrderCreationService) createOrder(ctx context.Context, order *models.Order) (uuid.UUID, error) {
	const op = "services.order.createOrder"

	orderUUID, err := os.orderCreator.Create(ctx, order)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %v", op, err)
	}

	order.OrderUUID = orderUUID
	for i := range order.Products {
		order.Products[i].OrderUUID = orderUUID
		order.TotalAmount += order.Products[i].Amount
	}

	return orderUUID, nil
}
