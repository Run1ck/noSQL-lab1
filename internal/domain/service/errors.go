package service

import "errors"

var (
	ErrInvalid      = errors.New("service id and name are required")
	ErrInvalidKind  = errors.New("invalid service kind")
	ErrInvalidDate  = errors.New("invalid date")
	ErrInvalidRange = errors.New("invalid date range")
	ErrNotFound     = errors.New("service not found")
)
