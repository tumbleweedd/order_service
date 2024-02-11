package models

import "github.com/google/uuid"

type Event interface {
	UUID() string
	Event() Event
}

type OrderStatus int

const (
	UndefinedStatus OrderStatus = iota
	OrderStatusCreated
	OrderStatusPaid
	OrderStatusDelivered
	OrderStatusCanceled
)

type Order struct {
	OrderUUID   uuid.UUID   `json:"order_uuid"`
	UserUUID    uuid.UUID   `json:"user_uuid"`
	Products    []Product   `json:"products"`
	Status      OrderStatus `json:"status"`
	PaymentType PaymentType `json:"payment_type"`
	TotalAmount uint64      `json:"total_amount"`
	WithPoints  int         `json:"with_points"`
}

type Product struct {
	UUID      uuid.UUID `json:"product_uuid"`
	OrderUUID uuid.UUID `json:"order_uuid"`
	Amount    uint64    `json:"amount"`
}

func (oe *Order) UUID() string {
	return oe.OrderUUID.String()
}

func (oe *Order) Event() Event {
	return oe
}

type StatusStruct struct {
	OrderUUID uuid.UUID   `json:"order_uuid"`
	Status    OrderStatus `json:"status"`
}

func (se *StatusStruct) UUID() string {
	return se.OrderUUID.String()
}

func (se *StatusStruct) Event() Event {
	return se
}

type PaymentType uint8

const (
	UndefinedType PaymentType = iota
	Card
	Points
)
