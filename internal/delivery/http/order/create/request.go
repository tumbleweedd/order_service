package create

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
)

var (
	errEmptyProducts = errors.New("products can't be empty")

	errInvalidPaymentType = errors.New("invalid payment type")
	errInvalidAmount      = errors.New("invalid price")
	errInvalidProductUUID = errors.New("invalid product_uuid")
	errInvalidUserUUID    = errors.New("invalid user_uuid")

	errIncorrectPointsValue = errors.New("incorrect points value")
)

type CreateOrderRequest struct {
	UserUUID    string     `json:"user_uuid"`
	Products    []Products `json:"products"`
	PaymentType string     `json:"payment_type"`
	WithPoints  int        `json:"with_points"`
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
		return errInvalidUserUUID
	}

	pType, ok := paymentTypes[req.PaymentType]
	if !ok {
		return errInvalidPaymentType
	}

	if len(req.Products) == 0 {
		return errEmptyProducts
	}

	var totalAmount uint64
	for _, product := range req.Products {
		if _, err = uuid.Parse(product.UUID); err != nil {
			return fmt.Errorf("%w: %s", errInvalidProductUUID, err.Error())
		}

		if product.Amount == 0 {
			return errInvalidAmount
		}

		totalAmount += product.Amount
	}

	if pType == models.Points {
		if req.WithPoints < 0 || req.WithPoints > int(totalAmount) {
			return errIncorrectPointsValue
		}
	}

	return nil
}

func (req *CreateOrderRequest) toDTO() models.Order {
	var products []models.Product
	for _, product := range req.Products {
		products = append(products, models.Product{
			UUID:   uuid.MustParse(product.UUID),
			Amount: product.Amount,
		})
	}

	return models.Order{
		Status:      models.OrderStatusCreated,
		UserUUID:    uuid.MustParse(req.UserUUID),
		PaymentType: paymentTypes[req.PaymentType],
		Products:    products,
		WithPoints:  req.WithPoints,
	}
}
