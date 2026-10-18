package request

import (
	"fmt"
	"strings"
)

const maxPurposeLen = 200
const maxParticipants = 500

type Details struct {
	Purpose        string
	Participants   int
	NeedsProjector bool
}

func NewDetails(purpose string, participants int, needsProjector bool) (*Details, error) {
	purpose = strings.TrimSpace(purpose)
	if purpose == "" || len([]rune(purpose)) > maxPurposeLen {
		return nil, fmt.Errorf("%w: purpose must be 1 to %d characters", ErrInvalidDetails, maxPurposeLen)
	}
	if participants < 1 || participants > maxParticipants {
		return nil, fmt.Errorf("%w: participants must be between 1 and %d", ErrInvalidDetails, maxParticipants)
	}
	return &Details{
		Purpose:        purpose,
		Participants:   participants,
		NeedsProjector: needsProjector,
	}, nil
}
