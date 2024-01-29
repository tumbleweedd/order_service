package models

type OrderStatus int

const (
	UndefinedStatus OrderStatus = iota
	OrderStatusCreated
	OrderStatusPaid
	OrderStatusDelivered
	OrderStatusCanceled
)

type Event interface {
	UUID() string
	Event() Event
}

type Order struct {
	OrderUUID   string      `json:"uuid"`
	UserUUID    string      `json:"user_uuid"`
	Products    []Product   `json:"products"`
	Status      OrderStatus `json:"status"`
	PaymentType PaymentType `json:"payment_type"`
	TotalAmount uint64      `json:"total_amount"`
	WithPoints  int         `json:"with_points"`
}

type Product struct {
	UUID      string `json:"uuid"`
	OrderUUID string `json:"order_uuid"`
	Amount    uint64 `json:"amount"`
}

func (oe *Order) UUID() string {
	return oe.OrderUUID
}

func (oe *Order) Event() Event {
	return oe
}

type StatusStruct struct {
	OrderUUID string      `json:"order_uuid"`
	Status    OrderStatus `json:"status"`
}

func (se *StatusStruct) UUID() string {
	return se.OrderUUID
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
