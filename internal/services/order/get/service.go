package get

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cacheImpl"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	internalErrors "github.com/tumbleweedd/two_services_system/order_service/internal/lib/errors"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type orderGetter interface {
	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) (ordersMap map[uuid.UUID]models.Order, err error)
	Order(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error)
}

type OrderRetrievalService struct {
	log   logger.Logger
	cache cacheImpl.CacheI[uuid.UUID, *models.Order]

	orderGetter orderGetter
}

func New(
	log logger.Logger,
	cache cacheImpl.CacheI[uuid.UUID, *models.Order],
	orderGetter orderGetter,
) *OrderRetrievalService {
	return &OrderRetrievalService{
		log:         log,
		cache:       cache,
		orderGetter: orderGetter,
	}
}

func (os *OrderRetrievalService) OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) ([]models.Order, error) {
	const op = "service.order.OrdersByUUIDs"

	result, notInCache := os.partitionOrdersByCache(ctx, UUIDs, op)

	if len(notInCache) == 0 {
		return result, nil
	}

	return os.fetchNotInCacheOrders(ctx, notInCache, result, op)
}

func (os *OrderRetrievalService) partitionOrdersByCache(ctx context.Context, UUIDs []uuid.UUID, op string) (result []models.Order, notInCache []uuid.UUID) {
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
		logger.Int("items in cache", len(result)),
		logger.Int("items not in cache", len(notInCache)),
	)

	return result, notInCache
}

func (os *OrderRetrievalService) checkCache(_ context.Context, orderUUID uuid.UUID,
	wg *sync.WaitGroup, inCacheCh chan models.Order, notInCacheCh chan uuid.UUID) {
	defer wg.Done()

	if value, ok := os.cache.Get(orderUUID); ok && value != nil {
		inCacheCh <- *value
		return
	}

	notInCacheCh <- orderUUID
}

func (os *OrderRetrievalService) fetchNotInCacheOrders(ctx context.Context, notInCache []uuid.UUID,
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
			os.cache.Add(order.OrderUUID, &order)
		}(order)
	}

	wg.Wait()

	os.log.InfoContext(ctx, op, logger.Int("orders from DB", len(ordersMap)))

	return result, nil
}

func (os *OrderRetrievalService) fetchOrdersFromDB(ctx context.Context, notInCache []uuid.UUID, op string) (map[uuid.UUID]models.Order, error) {
	ordersMap, err := os.orderGetter.OrdersByUUIDs(ctx, notInCache)
	if err != nil {
		if errors.Is(err, internalErrors.ErrOrderNotFound) {
			return nil, nil
		}

		os.log.Error(op, logger.String("get orders error", err.Error()))
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

func (os *OrderRetrievalService) OrderByUUID(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error) {
	const op = "service.order.Order"

	order, ok := os.cache.Get(orderUUID)
	if ok && order != nil {
		os.log.InfoContext(ctx, op, fmt.Sprint("cache was used"))
		return order, nil
	}

	return os.orderGetter.Order(ctx, orderUUID)
}
