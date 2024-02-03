package order_service_http

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	httpresponse "github.com/tumbleweedd/two_services_system/order_service/internal/lib/http"
	"net/http"
)

var (
	ErrEmptyOrderUUID = errors.New("order_uuid should not be empty")
)

type CancelOrderRequest struct {
	OrderUUID string `json:"order_uuid"`
}

func (r *CancelOrderRequest) validate() error {
	if r.OrderUUID == "" || len(r.OrderUUID) == 0 {
		return ErrEmptyOrderUUID
	}

	if _, err := uuid.Parse(r.OrderUUID); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidOrderUUID, err.Error())
	}

	return nil
}

func (r *CancelOrderRequest) toServiceRepresentation() uuid.UUID {
	return uuid.MustParse(r.OrderUUID)
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
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
	if err = h.orderService.Cancel(r.Context(), orderUUID); err != nil {
		h.log.Error("failed to cancel order: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(
		httpresponse.H{
			"message": "order canceled",
		},
	); err != nil {
		h.log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
