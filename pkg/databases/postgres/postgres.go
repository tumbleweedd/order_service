package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PgDB struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewPostgresDB(ctx context.Context, log *slog.Logger, dsn string) (*PgDB, error) {
	const op = "postgres.NewPostgresDB"

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pgDB := &PgDB{
		db:  db,
		log: log,
	}

	if err = pgDB.pingContext(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return pgDB, nil
}

func (pg *PgDB) GetDB() *sqlx.DB {
	return pg.db
}

func (pg *PgDB) Close() error {
	return pg.db.Close()
}

func (pg *PgDB) pingContext(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	status := "up"
	if err := pg.db.PingContext(ctx); err != nil {
		status = "down"
		pg.log.Error("database status", slog.String("status", status))
		return fmt.Errorf("failed to ping database: %w", err)
	}
	pg.log.Info("database status", slog.String("status", status))

	return nil
}
