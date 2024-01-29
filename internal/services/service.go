package services

import (
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
	orderCancaler OrderCancaler,
	orderEventsChan chan models.Event,
	statusEventChan chan models.Event,
) *Service {
	return &Service{
		log:          log,
		OrderService: NewOrderService(log, orderCreator, orderCancaler, orderEventsChan, statusEventChan),
	}
}
