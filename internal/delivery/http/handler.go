package order_service_http

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"log/slog"
	"net/http"
)

type Order interface {
	Create(ctx context.Context, order *models.Order) (string, error)
	Cancel(ctx context.Context, orderUUID uuid.UUID) error
	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) ([]models.Order, error)
}

type Handler struct {
	log *slog.Logger

	orderService Order
}

func NewHandler(log *slog.Logger, orderService Order) *Handler {
	return &Handler{
		log:          log,
		orderService: orderService,
	}
}

func (h *Handler) InitRoutes() http.Handler {
	mux := chi.NewRouter()

	mux.Route("/order", func(r chi.Router) {
		r.Post("/", h.createOrder)
		//r.Get("/{id}", h.getOrder)
		r.Post("/cancel", h.cancelOrder)
	})

	mux.Get("/orders", h.ordersByUUIDs)

	return mux
}
