package errors

import "errors"

var (
	ErrCancelOrderByStatus   = errors.New("order cannot be cancelled at this stage")
	ErrOrderNotFound         = errors.New("order not found")
	ErrOrderAlreadyCanceled  = errors.New("order already canceled")
	ErrOrderAlreadyDelivered = errors.New("order already delivered")
)
