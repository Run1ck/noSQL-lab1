package service

import "errors"

var (
	ErrInvalid = errors.New("service id and name are required")
)
