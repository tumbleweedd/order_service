package cancel

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	errEmptyOrderUUID   = errors.New("order_uuid should not be empty")
	errInvalidOrderUUID = errors.New("invalid order_uuid")
)

type CancelOrderRequest struct {
	OrderUUID string `json:"order_uuid"`
}

func (r *CancelOrderRequest) validate() error {
	if r.OrderUUID == "" || len(r.OrderUUID) == 0 {
		return errEmptyOrderUUID
	}

	if _, err := uuid.Parse(r.OrderUUID); err != nil {
		return fmt.Errorf("%w: %s", errInvalidOrderUUID, err.Error())
	}

	return nil
}

func (r *CancelOrderRequest) toServiceRepresentation() uuid.UUID {
	return uuid.MustParse(r.OrderUUID)
}
