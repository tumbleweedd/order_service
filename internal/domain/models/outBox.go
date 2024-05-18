package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type EventType string

const (
	OrderCreated  EventType = "ORDER_CREATED"
	OrderCanceled EventType = "ORDER_CANCELED"
)

type OutBoxMessage struct {
	ID        int             `json:"id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	Processed bool            `json:"processed"`
}

type OrderPayload struct {
	OrderUUID uuid.UUID `json:"order_uuid"`
}
