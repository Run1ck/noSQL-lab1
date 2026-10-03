package request

import "errors"

var (
	ErrEmptyCart        = errors.New("cannot create request from empty cart")
	ErrInvalidStatus    = errors.New("invalid status")
	ErrAlreadyProcessed = errors.New("request is already processed")
	ErrNotOwner         = errors.New("request belongs to another user")
	ErrSlotBooked       = errors.New("slot is already booked")
	ErrNotFound         = errors.New("request not found")
)
