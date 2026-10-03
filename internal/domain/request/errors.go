package request

import "errors"

var (
	ErrEmptyCart        = errors.New("cannot create request from empty cart")
	ErrInvalidApplicant = errors.New("applicant name and email are required")
	ErrInvalidStatus    = errors.New("invalid status")
	ErrAlreadyProcessed = errors.New("request is already processed")
)
