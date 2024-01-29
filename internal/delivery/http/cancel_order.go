package order_service_http

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
	httpresponse "github.com/tumbleweedd/two_services_system/order_service/internal/lib/http"
	"net/http"
)

type CancelOrderRequest struct {
	OrderUUID string `json:"order_uuid" validate:"required,uuid"`
}

func (cor *CancelOrderRequest) Validate() error {
	return validator.New().Struct(cor)
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	var request CancelOrderRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		h.log.Error("failed to decode request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = request.Validate(); err != nil {
		h.log.Error("failed to validate request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = h.orderService.Cancel(r.Context(), request.OrderUUID); err != nil {
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
