package services

import (
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/cache_imp"
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

	orderEventsChan chan models.Event,
	statusEventChan chan models.Event,
	done chan struct{},

	cache cache_imp.CacheI[uuid.UUID, *models.Order],
) *Service {
	return &Service{
		log: log,
		OrderService: NewOrderService(
			log,
			orderCreator,
			orderGetter,
			orderCancaler,
			orderEventsChan,
			statusEventChan,
			done,
			cache),
	}
}
