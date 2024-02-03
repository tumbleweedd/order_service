package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

	tx, err := or.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
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
		return uuid.Nil, fmt.Errorf("%s: order execute statement: %w", op, err)
	}

	var orderUUID uuid.UUID
	if err = row.Scan(&orderUUID); err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: scan result: %w", op, err)
	}

	const orderProductsQuery = `INSERT INTO "order_products" (order_uuid, product_uuid, amount) VALUES %s`
	var values []interface{}
	var placeholders []string

	for i, product := range order.Products {
		values = append(values, orderUUID, product.UUID, product.Amount)

		argId := i * 3

		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d)", argId+1, argId+2, argId+3))
	}

	fullQuery := fmt.Sprintf(orderProductsQuery, strings.Join(placeholders, ","))

	if _, err = tx.ExecContext(ctx, fullQuery, values...); err != nil {
		if rollBackErr := tx.Rollback(); rollBackErr != nil {
			or.log.Error(op, slog.String("error", rollBackErr.Error()))
			return uuid.Nil, fmt.Errorf("%s: rollback transaction: %w", op, rollBackErr)
		}
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: order_products execute statement: %w", op, err)
	}

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: event_uuid generate error: %w", op, err)
	}

	const outboxQuery = `INSERT INTO "outbox" (event_uuid, order_uuid) VALUES ($1, $2)`

	if _, err = tx.ExecContext(ctx, outboxQuery, eventUUID, orderUUID); err != nil {
		or.log.Error(op, slog.String("outbox insert error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: outbox insert error: %w", op, err)
	}

	if err = tx.Commit(); err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	return orderUUID, nil
}

func (or *OrderRepository) Cancel(ctx context.Context, orderUUID uuid.UUID) error {
	const op = "repository.order.Cancel"

	const query = `UPDATE "order" SET status = $1 WHERE uuid = $2`

	stmt, err := or.db.PreparexContext(ctx, query)
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

func (or *OrderRepository) OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) (map[uuid.UUID]models.Order, error) {
	const op = "repository.order.OrdersByUUIDs"

	ordersMap := make(map[uuid.UUID]models.Order, len(UUIDs))

	const orderQuery = `
							SELECT uuid, user_uuid, status, payment_type 
								FROM "order"
								WHERE uuid = ANY($1)
						`

	rows, err := or.db.QueryxContext(ctx, orderQuery, pq.Array(UUIDs))
	if err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var order models.Order
		if err = rows.Scan(&order.OrderUUID, &order.UserUUID, &order.Status, &order.PaymentType); err != nil {
			or.log.Error(op, slog.String("scan order error", err.Error()))
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		ordersMap[order.OrderUUID] = order
	}

	const orderProductsQuery = `
								SELECT order_uuid, product_uuid, amount
									FROM "order_products"
									WHERE order_uuid = ANY($1)
								`

	rows, err = or.db.QueryxContext(ctx, orderProductsQuery, pq.Array(UUIDs))
	if err != nil {
		or.log.Error(op, slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	for rows.Next() {
		var product models.Product
		if err = rows.Scan(&product.OrderUUID, &product.UUID, &product.Amount); err != nil {
			or.log.Error(op, slog.String("scan order_products ", err.Error()))
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		order := ordersMap[product.OrderUUID]
		order.Products = append(order.Products, product)

		ordersMap[product.OrderUUID] = order
	}

	return ordersMap, nil
}

func (or *OrderRepository) Status(ctx context.Context, orderUUID uuid.UUID) (int, error) {
	const op = "repository.order.Status"

	const query = `SELECT o.status FROM "order" o where o.uuid = $1`

	stmt, err := or.db.PreparexContext(ctx, query)
	if err != nil {
		or.log.Error(op, slog.String("prepare statement error", err.Error()))
		return 0, err
	}

	row := stmt.QueryRowxContext(ctx, orderUUID)

	var status int
	if err = row.Scan(&status); err != nil {
		or.log.Error(op, slog.String("scan status error", err.Error()))
		return 0, err
	}

	return status, nil
}
