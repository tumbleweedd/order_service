package services

import (
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cache_impl"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
)

type Service struct {
	log *slog.Logger

	*OrderService
}

func NewService(
	log *slog.Logger,

	orderCreator OrderCreator,
	orderGetter OrderGetter,
	orderCancaler OrderCancaler,

	cache cache_impl.CacheI[uuid.UUID, *models.Order],
) *Service {
	return &Service{
		log: log,
		OrderService: NewOrderService(
			log,
			orderCreator,
			orderGetter,
			orderCancaler,
			cache,
		),
	}
}
