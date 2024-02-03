package errors

import "errors"

var (
	ErrCancelOrderByStatus = errors.New("order cannot be cancelled at this stage")
)
