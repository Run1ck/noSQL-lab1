package service

import (
	"strings"
)

type Service struct {
	ID     string
	Name   string
	Kind   Kind
	Active bool
}

func New(id, name string, kind Kind, active bool) (*Service, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return nil, ErrInvalid
	}
	if _, err := ParseKind(string(kind)); err != nil {
		return nil, err
	}
	return &Service{
		ID:     id,
		Name:   name,
		Kind:   kind,
		Active: active,
	}, nil
}
