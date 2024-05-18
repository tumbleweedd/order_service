package cancel

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type orderCancaler interface {
	Cancel(ctx context.Context, orderUUID uuid.UUID) error
}

type Handler struct {
	log           logger.Logger
	orderCancaler orderCancaler
}

func NewHandler(log logger.Logger, orderCancaler orderCancaler) *Handler {
	return &Handler{
		log:           log,
		orderCancaler: orderCancaler,
	}
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	var request CancelOrderRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		h.log.Error("failed to decode request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = request.validate(); err != nil {
		h.log.Error("failed to validate request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	orderUUID := request.toServiceRepresentation()
	if err = h.orderCancaler.Cancel(r.Context(), orderUUID); err != nil {
		h.log.Error("failed to cancel order: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(
		map[string]string{
			"message": "order canceled",
		},
	); err != nil {
		h.log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
