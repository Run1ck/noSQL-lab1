package cart

import "errors"

var (
	ErrEmptySession = errors.New("session id is required")
	ErrInvalidQty   = errors.New("quantity must be positive")
	ErrItemNotFound = errors.New("item not found in cart")
)
