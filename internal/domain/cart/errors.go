package cart

import "errors"

var (
	ErrInvalidUser  = errors.New("user id is required")
	ErrItemNotFound = errors.New("item not found in cart")
	ErrInvalidUUID  = errors.New("invalid uuid")

	ErrPastDate           = errors.New("date is in the past")
	ErrServiceUnavailable = errors.New("service is not available")
	ErrSlotBooked         = errors.New("slot is already booked")
)
