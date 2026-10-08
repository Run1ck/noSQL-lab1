package cart

import "errors"

var (
	ErrItemNotFound = errors.New("item not found in cart")

	ErrPastDate           = errors.New("date is in the past")
	ErrServiceUnavailable = errors.New("service is not available")
	ErrSlotBooked         = errors.New("slot is already booked")
)
