package cacheImpl

import (
	"context"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"sync"
	"time"
)

type CacheI[K uuid.UUID, V *models.Order] interface {
	Get(key K) (value V, ok bool)
	Add(key K, value V)
}

type item[V any] struct {
	value  V
	expiry time.Time
}

func (i item[V]) isExpired() bool {
	return time.Now().After(i.expiry)
}

type Cache[K uuid.UUID, V *models.Order] struct {
	items      map[K]item[V]
	defaultTTL time.Duration
	mu         sync.RWMutex
}

func NewCache[K uuid.UUID, V *models.Order](ctx context.Context, defaultTTL time.Duration) *Cache[K, V] {
	cache := &Cache[K, V]{
		items:      make(map[K]item[V]),
		defaultTTL: defaultTTL,
	}

	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				cache.mu.Lock()

				for key, cacheItem := range cache.items {
					if cacheItem.isExpired() {
						delete(cache.items, key)
					}
				}

				cache.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	return cache
}

func (c *Cache[K, V]) Add(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = item[V]{
		value:  value,
		expiry: time.Now().Add(c.defaultTTL),
	}
}

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheItem, found := c.items[key]
	if !found {
		return cacheItem.value, false
	}

	if cacheItem.isExpired() {
		delete(c.items, key)
		return cacheItem.value, false
	}

	return cacheItem.value, true
}
