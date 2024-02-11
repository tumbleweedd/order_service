package cache_impl

import (
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
)

type CacheI[K uuid.UUID, V *models.Order] interface {
	Get(key K) (value V, ok bool)
	Add(key K, value V) (evicted bool)
}

type Cache struct {
	cache CacheI[uuid.UUID, *models.Order]
	log   *slog.Logger
}

func NewCache(
	cache CacheI[uuid.UUID, *models.Order],
	log *slog.Logger,
) *Cache {
	return &Cache{
		cache: cache,
		log:   log,
	}
}

func (c *Cache) Add(key uuid.UUID, value *models.Order) (evicted bool) {
	return c.cache.Add(key, value)
}

func (c *Cache) Get(key uuid.UUID) (value *models.Order, ok bool) {
	return c.cache.Get(key)
}
