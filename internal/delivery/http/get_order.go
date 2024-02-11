package order_service_http

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	httpresponse "github.com/tumbleweedd/two_services_system/order_service/internal/lib/http"
	"log/slog"
	"net/http"
)

var (
	ErrEmptyOrderIDs    = errors.New("no order ids passed")
	ErrInvalidOrderUUID = errors.New("invalid order_uuid")
)

type OrdersByUUIDsRequest struct {
	UUIDs []string `json:"uuids"`
}

func (r *OrdersByUUIDsRequest) validate() error {
	if len(r.UUIDs) == 0 {
		return ErrEmptyOrderIDs
	}

	for _, orderUUID := range r.UUIDs {
		if _, err := uuid.Parse(orderUUID); err != nil {
			return ErrInvalidOrderUUID
		}
	}

	return nil
}

func (r *OrdersByUUIDsRequest) toServiceRepresentation() []uuid.UUID {
	var result []uuid.UUID

	for _, orderUUID := range r.UUIDs {
		result = append(result, uuid.MustParse(orderUUID))
	}

	return result
}

func (h *Handler) ordersByUUIDs(w http.ResponseWriter, r *http.Request) {
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
	orders, err := h.orderService.OrdersByUUIDs(r.Context(), uuids)
	if err != nil {
		h.log.Error(op, slog.String("failed to get orders", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(
		httpresponse.H{
			"orders": orders,
		},
	); err != nil {
		h.log.Error(op, slog.String("failed to encode response", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
