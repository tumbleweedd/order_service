package get

import (
	"errors"

	"github.com/google/uuid"
)

var (
	errEmptyOrderIDs    = errors.New("no order ids passed")
	errInvalidOrderUUID = errors.New("invalid order_uuid")
)

type OrdersByUUIDsRequest struct {
	UUIDs []string `json:"uuids"`
}

func (r *OrdersByUUIDsRequest) validate() error {
	if len(r.UUIDs) == 0 {
		return errEmptyOrderIDs
	}

	for _, orderUUID := range r.UUIDs {
		if _, err := uuid.Parse(orderUUID); err != nil {
			return errInvalidOrderUUID
		}
	}

	return nil
}

func (r *OrdersByUUIDsRequest) toServiceRepresentation() []uuid.UUID {
	result := make([]uuid.UUID, 0, len(r.UUIDs))

	for _, orderUUID := range r.UUIDs {
		result = append(result, uuid.MustParse(orderUUID))
	}

	return result
}

type OrderByUUIDRequest struct {
	OrderUUID string `json:"order_uuid"`
}

func (r *OrderByUUIDRequest) validate() error {
	if _, err := uuid.Parse(r.OrderUUID); err != nil {
		return errInvalidOrderUUID
	}

	return nil
}

func (r *OrderByUUIDRequest) toServiceRepresentation() uuid.UUID {
	return uuid.MustParse(r.OrderUUID)
}
