package get

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
)

type orderGetter interface {
	OrdersByUUIDs(ctx context.Context, UUIDs []uuid.UUID) ([]models.Order, error)
	OrderByUUID(ctx context.Context, orderUUID uuid.UUID) (*models.Order, error)
}

type Handler struct {
	log *slog.Logger

	orderGetter orderGetter
}

func NewHandler(log *slog.Logger, orderGetter orderGetter) *Handler {
	return &Handler{
		log:         log,
		orderGetter: orderGetter,
	}
}

func (h *Handler) OrdersByUUIDs(w http.ResponseWriter, r *http.Request) {
	const op = "delivery.http.get_order.ordersByUUIDs"
	var request OrdersByUUIDsRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		h.log.Error(op, slog.String("failed to decode request", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = request.validate(); err != nil {
		h.log.Error(op, slog.String("failed to validate request", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	uuids := request.toServiceRepresentation()
	orders, err := h.orderGetter.OrdersByUUIDs(r.Context(), uuids)
	if err != nil {
		h.log.Error(op, slog.String("failed to get orders", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(
		map[string]interface{}{
			"orders": orders,
		},
	); err != nil {
		h.log.Error(op, slog.String("failed to encode response", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
