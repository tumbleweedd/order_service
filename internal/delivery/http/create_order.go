package order_service_http

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	"net/http"

	httpresponse "github.com/tumbleweedd/two_services_system/order_service/internal/lib/http"
)

var (
	ErrEmptyProducts = errors.New("products can't be empty")

	ErrInvalidPaymentType = errors.New("invalid payment type")
	ErrInvalidAmount      = errors.New("invalid price")
	ErrInvalidProductUUID = errors.New("invalid order_uuid_id")
	ErrInvalidUserUUID    = errors.New("invalid user_uuid_id")

	ErrIncorrectPointsValue = errors.New("incorrect points value")
)

type CreateOrderRequest struct {
	UserUUID    string     `json:"user_uuid" validate:"required,uuid"`
	Products    []Products `json:"products" validate:"required"`
	PaymentType string     `json:"payment_type"`
	WithPoints  int        `json:"with_points" validate:"gte=0"`
}

type Products struct {
	UUID   string `json:"uuid"`
	Amount uint64 `json:"amount"`
}

var paymentTypes = map[string]models.PaymentType{
	"card":   models.Card,
	"points": models.Points,
}

func (req *CreateOrderRequest) validate() error {
	_, err := uuid.Parse(req.UserUUID)
	if err != nil {
		return ErrInvalidUserUUID
	}

	pType, ok := paymentTypes[req.PaymentType]
	if !ok {
		return ErrInvalidPaymentType
	}

	if len(req.Products) == 0 {
		return ErrEmptyProducts
	}

	var totalAmount uint64
	for _, product := range req.Products {
		if _, err = uuid.Parse(product.UUID); err != nil {
			return ErrInvalidProductUUID
		}

		if product.Amount == 0 {
			return ErrInvalidAmount
		}

		totalAmount += product.Amount
	}

	if pType == models.Points {
		if req.WithPoints < 0 || req.WithPoints > int(totalAmount) {
			return ErrIncorrectPointsValue
		}
	}

	return nil
}

func (req *CreateOrderRequest) ToDTO() models.Order {
	var products []models.Product
	for _, product := range req.Products {
		products = append(products, models.Product{
			UUID:   product.UUID,
			Amount: product.Amount,
		})
	}

	return models.Order{
		Status:      models.OrderStatusCreated,
		UserUUID:    req.UserUUID,
		PaymentType: paymentTypes[req.PaymentType],
		Products:    products,
		WithPoints:  req.WithPoints,
	}
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
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

	order := request.ToDTO()
	orderUUID, err := h.orderService.Create(
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
		httpresponse.H{
			"order_uuid": orderUUID,
		},
	); err != nil {
		h.log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
