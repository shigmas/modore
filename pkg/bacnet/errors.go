package bacnet

import (
	"errors"
)

var (
	ErrInvalidData      = errors.New("invalid data")
	ErrInsufficientData = errors.New("unexpected end of data")
	ErrValueTooLarge    = errors.New("value too large for context")
	ErrNotImplemented   = errors.New("not implemented")
)
