package cancel

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cacheImpl"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	internalErrors "github.com/tumbleweedd/two_services_system/order_service/internal/lib/errors"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type orderGetter interface {
	Order(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error)
}

type orderCancaler interface {
	Cancel(ctx context.Context, orderUUID uuid.UUID) error
}

type OrderCancellationService struct {
	log   logger.Logger
	cache cacheImpl.CacheI[uuid.UUID, *models.Order]

	orderCancaler orderCancaler
	orderGetter   orderGetter
}

func New(
	log logger.Logger,
	cache cacheImpl.CacheI[uuid.UUID, *models.Order],
	orderCancaler orderCancaler,
	orderGetter orderGetter,
) *OrderCancellationService {
	return &OrderCancellationService{
		log:           log,
		cache:         cache,
		orderCancaler: orderCancaler,
		orderGetter:   orderGetter,
	}
}

func (os *OrderCancellationService) Cancel(ctx context.Context, orderUUID uuid.UUID) (err error) {
	const op = "services.order.Cancel"

	var needUpdateCache bool

	order, exist := os.cache.Get(orderUUID)
	if !exist {
		order, err = os.orderGetter.Order(ctx, orderUUID)
		if err != nil {
			os.log.Error(op, logger.String("get order error", err.Error()))
			return fmt.Errorf("%s: %w", op, err)
		}

		needUpdateCache = true
	}

	for _, product := range order.Products {
		order.TotalAmount += product.Amount
	}

	defer func() {
		if needUpdateCache {
			os.cache.Add(orderUUID, order)
			os.log.InfoContext(ctx, op, "cache was updated")
		}
	}()

	switch order.Status {
	case models.OrderStatusCreated, models.OrderStatusPaid:
		if err = os.orderCancaler.Cancel(ctx, orderUUID); err != nil {
			if errors.Is(err, internalErrors.ErrOrderNotFound) {
				os.log.Error(op, logger.String("order not found by uuid", err.Error()))
				return fmt.Errorf("%s, order not found: %w", op, err)
			}
			os.log.Error(op, logger.String("cancel order error", err.Error()))
			return fmt.Errorf("%s, cancel order: %w", op, err)
		}

		order.Status = models.OrderStatusCanceled

		needUpdateCache = true
	case models.OrderStatusCanceled:
		return internalErrors.ErrOrderAlreadyCanceled
	case models.OrderStatusDelivered:
		return internalErrors.ErrOrderAlreadyDelivered
	default:
		return fmt.Errorf("order cancellation error by status %v: %w", order.Status, err)
	}

	return
}
