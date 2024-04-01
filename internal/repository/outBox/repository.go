package outBox

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type Repository struct {
	db *sqlx.DB

	log logger.Logger
}

func New(log logger.Logger, db *sqlx.DB) *Repository {
	return &Repository{db: db, log: log}
}

type orderCreatedData struct {
	OrderUUID uuid.UUID `json:"order_uuid"`
}

func (r *Repository) Insert(ctx context.Context, orderUUID uuid.UUID, eventType models.EventType) error {
	const op = "Repository.Insert"

	payload := orderCreatedData{OrderUUID: orderUUID}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		r.log.Error(op, logger.String("json marshal error", err.Error()))
		return fmt.Errorf("%s: json marshal error: %w", op, err)
	}

	const outboxQuery = `INSERT INTO "outbox" (event_type, payload) VALUES ($1, $2)`

	if _, err = r.db.ExecContext(ctx, outboxQuery, eventType, jsonData); err != nil {
		r.log.Error(op, logger.String("outbox insert error", err.Error()))
		return fmt.Errorf("%s: outbox insert error: %w", op, err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, eventIDs []int) error {
	const op = "Repository.Delete"

	const outboxQuery = `DELETE FROM "outbox" WHERE id = ANY($1)`

	if _, err := r.db.ExecContext(ctx, outboxQuery, pq.Array(eventIDs)); err != nil {
		r.log.Error(op, logger.String("outbox delete error", err.Error()))
		return fmt.Errorf("%s: outbox delete error: %w", op, err)
	}

	return nil
}

func (r *Repository) FetchUnprocessedMessages(ctx context.Context) (messages []models.OutBoxMessage, err error) {
	const op = "Repository.FetchUnprocessedMessages"

	const outboxQuery = `
							SELECT id, event_type, payload, processed 
								FROM "outbox" 
								WHERE processed = false
								ORDER BY created_at DESC
								LIMIT 100
						`

	rows, err := r.db.QueryContext(ctx, outboxQuery)
	if err != nil {
		return nil, fmt.Errorf("query outbox: %w", err)
	}
	defer func() {
		closeErr := rows.Close()
		if err != nil {
			if closeErr != nil {
				r.log.Error(op, logger.String("error", closeErr.Error()))
			}
			return
		}
		err = closeErr
	}()

	for rows.Next() {
		var msg models.OutBoxMessage
		if err = rows.Scan(&msg.ID, &msg.EventType, &msg.Payload, &msg.Processed); err != nil {
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
