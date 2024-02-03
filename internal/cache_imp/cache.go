package cache_imp

import (
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
)

type CacheI[K uuid.UUID, V *models.Order] interface {
	Get(key K) (value V, ok bool)
	Add(key K, value V) (evicted bool)
}

type Cache struct {
	//orderRepository OrderRepository
	cache CacheI[uuid.UUID, *models.Order]
	//log             *slog.Logger
}

//type OrderRepository interface {
//	Create(ctx context.Context, order *models.Order) (uuid.UUID, error)
//	Cancel(ctx context.Context, orderUUID uuid.UUID) error
//	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) (ordersMap map[uuid.UUID]models.Order, err error)
//	Status(ctx context.Context, orderUUID uuid.UUID) (int, error)
//}

func NewCache(
	//orderRepository OrderRepository,
	cache CacheI[uuid.UUID, *models.Order],
	// log *slog.Logger,
) *Cache {
	return &Cache{
		cache: cache,
		//orderRepository: orderRepository,
		//log:             log,
	}
}

func (c *Cache) Add(key uuid.UUID, value *models.Order) (evicted bool) {
	return c.cache.Add(key, value)
	//const op = "cache_imp.cache.Add"
	//orderUUID, err := c.orderRepository.Create(ctx, order)
	//if err != nil {
	//	return "", fmt.Errorf("%s: %v", op, err)
	//}
	//
	//order.OrderUUID = orderUUID
	//for i := range order.Products {
	//	order.Products[i].OrderUUID = orderUUID
	//	order.TotalAmount += order.Products[i].Amount
	//}
	//
	//if evicted := c.cache.Add(orderUUID, order); evicted {
	//	c.log.WarnContext(ctx, op, fmt.Sprint("cache size was exceeded"))
	//}
	//
	//c.log.InfoContext(ctx, op, fmt.Sprint("cache was updated"))
	//
	//return orderUUID.String(), nil
}

func (c *Cache) Get(key uuid.UUID) (value *models.Order, ok bool) {
	return c.cache.Get(key)
}
