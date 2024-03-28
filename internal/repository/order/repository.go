package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	internalErrors "github.com/tumbleweedd/two_services_system/order_service/internal/lib/errors"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type outBoxRepository interface {
	Insert(ctx context.Context, orderID uuid.UUID) error
}

type Repository struct {
	log              logger.Logger
	db               *sqlx.DB
	outBoxRepository outBoxRepository
}

func NewOrderRepository(log logger.Logger, db *sqlx.DB, outBoxRepository outBoxRepository) *Repository {
	return &Repository{
		log:              log,
		db:               db,
		outBoxRepository: outBoxRepository,
	}
}

func (or *Repository) Create(
	ctx context.Context,
	order *models.Order,
) (orderUUID uuid.UUID, err error) {
	const op = "repository.order.Create"

	tx, err := or.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: begin transaction: %w", op, err)
	}

	defer func() {
		if err != nil {
			if rollBackErr := tx.Rollback(); rollBackErr != nil {
				or.log.Error(op, logger.String("error", rollBackErr.Error()))
				errors.Join(err, fmt.Errorf("%s: rollback transaction: %w", op, rollBackErr))
			}
		}
	}()

	const orderQuery = `INSERT INTO "order" (user_uuid, status, payment_type) VALUES ($1, $2, $3) RETURNING uuid`

	row := tx.QueryRowContext(ctx, orderQuery, order.UserUUID, order.Status, order.PaymentType)
	if err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: order execute statement: %w", op, err)
	}

	if err = row.Scan(&orderUUID); err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
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
		or.log.Error(op, logger.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: order_products execute statement: %w", op, err)
	}

	if err = or.outBoxRepository.Insert(ctx, orderUUID); err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: outbox insert error: %w", op, err)
	}
	//eventUUID, err := uuid.NewUUID()
	//if err != nil {
	//	or.log.Error(op, logger.String("error", err.Error()))
	//	return uuid.Nil, fmt.Errorf("%s: event_uuid generate error: %w", op, err)
	//}
	//
	//const outboxQuery = `INSERT INTO "outbox" (event_uuid, order_uuid) VALUES ($1, $2)`
	//
	//if _, err = tx.ExecContext(ctx, outboxQuery, eventUUID, orderUUID); err != nil {
	//	or.log.Error(op, logger.String("outbox insert error", err.Error()))
	//	return uuid.Nil, fmt.Errorf("%s: outbox insert error: %w", op, err)
	//}

	if err = tx.Commit(); err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	return
}

func (or *Repository) Cancel(ctx context.Context, orderUUID uuid.UUID) (err error) {
	const op = "repository.order.Cancel"

	tx, err := or.db.Begin()
	if err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return fmt.Errorf("%s: begin transaction: %w", op, err)
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	const cancelQuery = `UPDATE "order" SET status = $1 WHERE uuid = $2`

	_, err = tx.ExecContext(ctx, cancelQuery, int(models.OrderStatusCanceled), orderUUID)
	if err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	const outboxQuery = `INSERT INTO "outbox" (event_uuid, order_uuid) VALUES ($1, $2)`

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		or.log.Error(op, logger.String("outbox insert error", err.Error()))
		return fmt.Errorf("%s: outbox insert error: %w", op, err)
	}

	if _, err = tx.ExecContext(ctx, outboxQuery, eventUUID, orderUUID); err != nil {
		or.log.Error(op, logger.String("outbox insert error", err.Error()))
		return fmt.Errorf("%s: outbox insert error: %w", op, err)
	}

	return tx.Commit()
}

func (or *Repository) OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) (map[uuid.UUID]models.Order, error) {
	const op = "repository.order.OrdersByUUIDs"

	ordersMap := make(map[uuid.UUID]models.Order, len(UUIDs))

	const orderQuery = `
							SELECT uuid, user_uuid, status, payment_type 
								FROM "order"
								WHERE uuid = ANY($1)
						`

	rows, err := or.db.QueryContext(ctx, orderQuery, pq.Array(UUIDs))
	if err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var order models.Order
		if err = rows.Scan(&order.OrderUUID, &order.UserUUID, &order.Status, &order.PaymentType); err != nil {
			or.log.Error(op, logger.String("scan order error", err.Error()))
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		ordersMap[order.OrderUUID] = order
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(ordersMap) == 0 {
		return nil, internalErrors.ErrOrderNotFound
	}

	const orderProductsQuery = `
								SELECT order_uuid, product_uuid, amount
									FROM "order_products"
									WHERE order_uuid = ANY($1)
								`

	rows, err = or.db.QueryContext(ctx, orderProductsQuery, pq.Array(UUIDs))
	if err != nil {
		or.log.Error(op, logger.String("error", err.Error()))
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	for rows.Next() {
		var product models.Product
		if err = rows.Scan(&product.OrderUUID, &product.UUID, &product.Amount); err != nil {
			or.log.Error(op, logger.String("scan order_products ", err.Error()))
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		order := ordersMap[product.OrderUUID]
		order.Products = append(order.Products, product)

		ordersMap[product.OrderUUID] = order
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return ordersMap, nil
}

func (or *Repository) Status(ctx context.Context, orderUUID uuid.UUID) (int, error) {
	const op = "repository.order.Status"

	const query = `SELECT o.status FROM "order" o where o.uuid = $1`

	stmt, err := or.db.PrepareContext(ctx, query)
	if err != nil {
		or.log.Error(op, logger.String("prepare statement error", err.Error()))
		return 0, err
	}

	row := stmt.QueryRowContext(ctx, orderUUID)

	var status int
	if err = row.Scan(&status); err != nil {
		or.log.Error(op, slog.String("scan status error", err.Error()))
		return 0, err
	}

	return status, nil
}

func (or *Repository) Order(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error) {
	op := "repository.order.Order"

	const orderQuery = `SELECT o.uuid, o.user_uuid, o.status, o.payment_type FROM "order" o where o.uuid = $1`

	stmt, err := or.db.PrepareContext(ctx, orderQuery)
	if err != nil {
		or.log.Error(op, slog.String("prepare statement error", err.Error()))
		return nil, err
	}

	row := stmt.QueryRowContext(ctx, orderUUID)

	var order models.Order
	if err = row.Scan(&order.OrderUUID, &order.UserUUID, &order.Status, &order.PaymentType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalErrors.ErrOrderNotFound
		}
		or.log.Error(op, slog.String("scan order error", err.Error()))
		return nil, err
	}

	const orderProductsQuery = `
									SELECT op.order_uuid, op.product_uuid, op.amount 
										FROM "order_products" op 
										WHERE op.order_uuid = $1
								`

	rows, err := or.db.QueryContext(ctx, orderProductsQuery, orderUUID)
	if err != nil {
		or.log.Error(op, slog.String("execute statement error", err.Error()))
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	for rows.Next() {
		var product models.Product
		if err = rows.Scan(&product.OrderUUID, &product.UUID, &product.Amount); err != nil {
			or.log.Error(op, slog.String("scan order_products ", err.Error()))
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		order.Products = append(order.Products, product)
	}

	return &order, nil
}
