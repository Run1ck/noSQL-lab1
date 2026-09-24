package request

import "fmt"

type Status string

const (
	StatusNew      Status = "new"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusNew, StatusApproved, StatusRejected:
		return Status(s), nil
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidStatus, s)
}

func (s Status) IsFinal() bool {
	return s != StatusNew
}
