package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
	"strings"
)

type OrderRepository struct {
	log *slog.Logger
	db  *sqlx.DB
}

func NewOrderRepository(log *slog.Logger, db *sqlx.DB) *OrderRepository {
	return &OrderRepository{
		log: log,
		db:  db,
	}
}

func (or *OrderRepository) Create(
	ctx context.Context,
	order *models.Order,
) (uuid.UUID, error) {
	const op = "repository.order.Create"

	tx, err := or.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: begin transaction: %w", op, err)
	}

	const orderQuery = `INSERT INTO "order" (user_uuid, status, payment_type) VALUES ($1, $2, $3) RETURNING uuid`

	row := tx.QueryRowContext(ctx, orderQuery, order.UserUUID, order.Status, order.PaymentType)
	if err != nil {
		if rollBackErr := tx.Rollback(); rollBackErr != nil {
			or.log.Error(op, slog.String("error", rollBackErr.Error()))
			return uuid.Nil, fmt.Errorf("%s: rollback transaction: %w", op, rollBackErr)
		}
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	var orderUUID uuid.UUID
	if err = row.Scan(&orderUUID); err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: scan result: %w", op, err)
	}

	const orderProductsQuery = `INSERT INTO "order_products" (order_uuid, product_uuid, amount) VALUES %s`
	var values []interface{}
	var placeholders []string
	argId := 0

	for _, product := range order.Products {
		values = append(values, orderUUID, product.UUID, product.Amount)
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d)", argId+1, argId+2, argId+3))
	}

	fullQuery := fmt.Sprintf(orderProductsQuery, strings.Join(placeholders, ","))

	if _, err = tx.ExecContext(ctx, fullQuery, values...); err != nil {
		if rollBackErr := tx.Rollback(); rollBackErr != nil {
			or.log.Error(op, slog.String("error", rollBackErr.Error()))
			return uuid.Nil, fmt.Errorf("%s: rollback transaction: %w", op, rollBackErr)
		}
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	if err = tx.Commit(); err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	return orderUUID, nil
}

func (or *OrderRepository) Cancel(ctx context.Context, orderUUID string) error {
	const op = "repository.order.Cancel"

	const query = `UPDATE "order" SET status = $1 WHERE uuid = $2`

	stmt, err := or.db.PrepareContext(ctx, query)
	if err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	_, err = stmt.ExecContext(ctx, int(models.OrderStatusCanceled), orderUUID)
	if err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return nil
}
