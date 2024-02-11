package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cache_impl"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	internal_errors "github.com/tumbleweedd/two_services_system/order_service/internal/lib/errors"
	"log/slog"
	"sync"
)

type OrderService struct {
	log *slog.Logger

	orderCreator  OrderCreator
	orderGetter   OrderGetter
	orderCancaler OrderCancaler

	cache cache_impl.CacheI[uuid.UUID, *models.Order]
}

type OrderCreator interface {
	Create(ctx context.Context, order *models.Order) (uuid.UUID, error)
}

type OrderGetter interface {
	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) (ordersMap map[uuid.UUID]models.Order, err error)
	Order(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error)
}

type OrderCancaler interface {
	Cancel(ctx context.Context, orderUUID uuid.UUID) error
}

func NewOrderService(
	log *slog.Logger,

	orderCreator OrderCreator,
	orderGetter OrderGetter,
	orderCancaler OrderCancaler,

	cache cache_impl.CacheI[uuid.UUID, *models.Order],
) *OrderService {
	return &OrderService{
		log:           log,
		orderCreator:  orderCreator,
		orderGetter:   orderGetter,
		orderCancaler: orderCancaler,
		cache:         cache,
	}
}

func (os *OrderService) Create(ctx context.Context, order *models.Order) (string, error) {
	const op = "services.order.Create"

	orderUUID, err := os.createOrder(ctx, order)
	if err != nil {
		return "", fmt.Errorf("%s: %v", op, err)
	}

	_ = os.cache.Add(orderUUID, order)

	os.log.InfoContext(ctx, op, fmt.Sprint("cache was updated"))

	return orderUUID.String(), nil
}

func (os *OrderService) createOrder(ctx context.Context, order *models.Order) (uuid.UUID, error) {
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

func (os *OrderService) Cancel(ctx context.Context, orderUUID uuid.UUID) (err error) {
	const op = "services.order.Cancel"

	var needUpdateCache bool

	order, exist := os.cache.Get(orderUUID)
	if !exist {
		order, err = os.orderGetter.Order(ctx, orderUUID)
		if err != nil {
			os.log.Error(op, slog.String("get order error", err.Error()))
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
			if errors.Is(err, internal_errors.ErrOrderNotFound) {
				os.log.Error(op, slog.String("order not found by uuid", err.Error()))
				return fmt.Errorf("%s, order not found: %w", op, err)
			}
			os.log.Error(op, slog.String("cancel order error", err.Error()))
			return fmt.Errorf("%s, cancel order: %w", op, err)
		}

		order.Status = models.OrderStatusCanceled

		needUpdateCache = true
	case models.OrderStatusCanceled:
		return internal_errors.ErrOrderAlreadyCanceled
	case models.OrderStatusDelivered:
		return internal_errors.ErrOrderAlreadyDelivered
	default:
		return fmt.Errorf("order cancellation error by status %v: %w", order.Status, err)
	}

	return
}

func (os *OrderService) OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) ([]models.Order, error) {
	const op = "service.order.OrdersByUUIDs"

	result, notInCache := os.partitionOrdersByCache(ctx, UUIDs, op)

	if len(notInCache) == 0 {
		return result, nil
	}

	return os.fetchNotInCacheOrders(ctx, notInCache, result, op)
}

func (os *OrderService) partitionOrdersByCache(ctx context.Context, UUIDs []uuid.UUID, op string) (result []models.Order, notInCache []uuid.UUID) {
	inCacheCh := make(chan models.Order, len(UUIDs))
	notInCacheCh := make(chan uuid.UUID, len(UUIDs))
	wg := sync.WaitGroup{}

	for _, id := range UUIDs {
		wg.Add(1)
		go os.checkCache(ctx, id, &wg, inCacheCh, notInCacheCh)
	}

	wg.Wait()
	close(inCacheCh)
	close(notInCacheCh)

	result = make([]models.Order, 0, len(UUIDs))
	for order := range inCacheCh {
		result = append(result, order)
	}

	notInCache = make([]uuid.UUID, 0, len(UUIDs))
	for orderUUID := range notInCacheCh {
		notInCache = append(notInCache, orderUUID)
	}

	os.log.InfoContext(ctx, op,
		slog.Int("items in cache", len(result)),
		slog.Int("items not in cache", len(notInCache)),
	)

	return result, notInCache
}

func (os *OrderService) checkCache(_ context.Context, orderUUID uuid.UUID,
	wg *sync.WaitGroup, inCacheCh chan models.Order, notInCacheCh chan uuid.UUID) {
	defer wg.Done()

	if value, ok := os.cache.Get(orderUUID); ok && value != nil {
		inCacheCh <- *value
		return
	}

	notInCacheCh <- orderUUID
}

func (os *OrderService) fetchNotInCacheOrders(ctx context.Context, notInCache []uuid.UUID,
	result []models.Order, op string) ([]models.Order, error) {
	ordersMap, err := os.fetchOrdersFromDB(ctx, notInCache, op)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	for _, order := range ordersMap {
		result = append(result, order)

		wg.Add(1)
		go func(order models.Order) {
			defer wg.Done()
			_ = os.cache.Add(order.OrderUUID, &order)
		}(order)
	}

	wg.Wait()

	os.log.InfoContext(ctx, op, slog.Int("orders from DB", len(ordersMap)))

	return result, nil
}

func (os *OrderService) fetchOrdersFromDB(ctx context.Context, notInCache []uuid.UUID, op string) (map[uuid.UUID]models.Order, error) {
	ordersMap, err := os.orderGetter.OrdersByUUIDs(ctx, notInCache)
	if err != nil {
		if errors.Is(err, internal_errors.ErrOrderNotFound) {
			return nil, nil
		}

		os.log.Error(op, slog.String("get orders error", err.Error()))
		return nil, err
	}

	for orderUUID, order := range ordersMap {
		for _, product := range order.Products {
			order.TotalAmount += product.Amount
		}
		ordersMap[orderUUID] = order
	}

	return ordersMap, nil
}
