package create

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"github.com/tumbleweedd/two_services_system/order_service/pkg/logger"
)

type orderCreator interface {
	Create(ctx context.Context, order *models.Order) (string, error)
}

type Handler struct {
	log logger.Logger

	orderCreator orderCreator
}

func NewHandler(log logger.Logger, orderCreator orderCreator) *Handler {
	return &Handler{
		log:          log,
		orderCreator: orderCreator,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request CreateOrderRequest

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

	order := request.toDTO()
	orderUUID, err := h.orderCreator.Create(
		r.Context(),
		&order,
	)
	if err != nil {
		h.log.Error("failed to create order: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(
		map[string]string{
			"order_uuid": orderUUID,
		},
	); err != nil {
		h.log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
