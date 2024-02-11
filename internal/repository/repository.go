package repository

import (
	"github.com/jmoiron/sqlx"
	"log/slog"
)

type Repository struct {
	log *slog.Logger

	*OrderRepository
}

func NewRepository(log *slog.Logger, db *sqlx.DB) *Repository {
	return &Repository{
		log:             log,
		OrderRepository: NewOrderRepository(log, db),
	}
}
