package services

import (
	"context"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
)

type cancellation interface {
	Cancel(ctx context.Context, orderUUID uuid.UUID) (err error)
}

type retrieval interface {
	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) ([]models.Order, error)
	OrderByUUID(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error)
}

type creation interface {
	Create(ctx context.Context, order *models.Order) (string, error)
}

type Service struct {
	log *slog.Logger

	cancellation
	retrieval
	creation
}

func NewService(
	log *slog.Logger,

	orderCreation creation,
	orderRetrieval retrieval,
	orderCancellation cancellation,
) *Service {
	return &Service{
		log:          log,
		creation:     orderCreation,
		retrieval:    orderRetrieval,
		cancellation: orderCancellation,
	}
}
