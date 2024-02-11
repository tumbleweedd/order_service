package outbox_producer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/tumbleweedd/two_services_system/order_service/internal/config"
	"log/slog"
)

type OutboxProducer struct {
	producer    sarama.SyncProducer
	db          *sqlx.DB
	kafkaConfig config.KafkaConfig
	log         *slog.Logger
}

type outboxMessage struct {
	EventUUID uuid.UUID `json:"event_uuid"`
	OrderUUID uuid.UUID `json:"order_uuid"`
}

func New(
	producer sarama.SyncProducer,
	db *sqlx.DB,
	kafkaConfig config.KafkaConfig,
	log *slog.Logger,
) *OutboxProducer {
	return &OutboxProducer{
		producer:    producer,
		db:          db,
		kafkaConfig: kafkaConfig,
		log:         log,
	}
}

const messageSendLimit = 100

func (op *OutboxProducer) ProduceMessages(ctx context.Context) (err error) {
	tx, err := op.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	defer func() {
		if err != nil {
			if rollBackErr := tx.Rollback(); rollBackErr != nil {
				op.log.Error("outbox_producer", slog.String("error", rollBackErr.Error()))
				err = errors.Join(err, rollBackErr)
			}
		}
	}()

	const outboxSelectQuery = `
								SELECT event_uuid, order_uuid 
									FROM "outbox"
									WHERE send = FALSE
									ORDER BY order_uuid
									LIMIT 100
								`

	rows, err := tx.QueryContext(ctx, outboxSelectQuery)
	if err != nil {
		op.log.Error("outbox_producer", slog.String("error", err.Error()))
		return fmt.Errorf("query outbox: %w", err)
	}
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			op.log.Error("outbox_producer", slog.String("error", err.Error()))
		}
	}(rows)

	var eventUUIDs []uuid.UUID
	saramaMessages := make([]*sarama.ProducerMessage, 0, messageSendLimit)

	for rows.Next() {
		msg := outboxMessage{}
		if err = rows.Scan(&msg.EventUUID, &msg.OrderUUID); err != nil {
			op.log.Error("outbox_producer", slog.String("error", err.Error()))
			return fmt.Errorf("scan outbox: %w", err)
		}

		bytes, err := json.Marshal(msg)
		if err != nil {
			op.log.Error("outbox_producer", slog.String("error", err.Error()))
			return fmt.Errorf("marshal outbox: %w", err)
		}

		saramaMessages = append(saramaMessages, &sarama.ProducerMessage{
			Topic: op.kafkaConfig.OrderEventTopic,
			Value: sarama.ByteEncoder(bytes),
		})

		eventUUIDs = append(eventUUIDs, msg.EventUUID)
	}

	const outboxUpdateQuery = `UPDATE "outbox" SET send = TRUE WHERE event_uuid = ANY($1)`

	// Сначала обновляем данные в таблице, чтобы в случае если база упадёт,
	// транзакция отменилась, и мы не отправили сообщения в топик
	//
	// При обратной последовательности может произойти так, что, сначала
	// отправляя сообщения в топик, а после обновляя данные в таблице, мы
	// база может упасть, а сообщения уже будут отправлены.
	if _, err = tx.ExecContext(ctx, outboxUpdateQuery, pq.Array(eventUUIDs)); err != nil {
		op.log.Error("outbox_producer", slog.String("error", err.Error()))
		return fmt.Errorf("update outbox: %w", err)
	}

	if err = op.producer.SendMessages(saramaMessages); err != nil {
		op.log.Error("outbox_producer", slog.String("error", err.Error()))
		return fmt.Errorf("send messages: %w", err)
	}

	return tx.Commit()
}
