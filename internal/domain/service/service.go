package service

import "strings"

type Service struct {
	ID     string
	Name   string
	Active bool
}

func New(id, name string, active bool) (*Service, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return nil, ErrInvalid
	}
	return &Service{
		ID:     id,
		Name:   name,
		Active: active,
	}, nil
}
