package outBox

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type Repository struct {
	db *sqlx.DB

	log logger.Logger
}

func New(log logger.Logger, db *sqlx.DB) *Repository {
	return &Repository{db: db, log: log}
}

func (or *Repository) Insert(ctx context.Context, orderUUID uuid.UUID) error {
	const op = "Repository.Insert"

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return fmt.Errorf("%s: event_uuid generate error: %w", op, err)
	}

	const outboxQuery = `INSERT INTO "outbox" (event_uuid, order_uuid) VALUES ($1, $2)`

	if _, err = or.db.ExecContext(ctx, outboxQuery, eventUUID, orderUUID); err != nil {
		or.log.Error(op, logger.String("outbox insert error", err.Error()))
		return fmt.Errorf("%s: outbox insert error: %w", op, err)
	}

	return nil
}
