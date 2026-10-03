package service

import "fmt"

type Kind string

const (
	Room      Kind = "room"
	Lab       Kind = "lab"
	Equipment Kind = "equipment"
)

func ParseKind(s string) (Kind, error) {
	switch Kind(s) {
	case Room, Lab, Equipment:
		return Kind(s), nil
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidKind, s)
}
