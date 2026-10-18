package request_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"booking/internal/domain/request"
)

func Test_NewDetails(t *testing.T) {
	tests := []struct {
		name         string
		purpose      string
		participants int
		wantErr      bool
	}{
		{"ok, trims spaces", "  Семинар по БД  ", 25, false},
		{"200 cyrillic chars", strings.Repeat("я", 200), 1, false},
		{"empty purpose", "   ", 25, true},
		{"201 chars", strings.Repeat("я", 201), 25, true},
		{"zero participants", "Семинар", 0, true},
		{"too many participants", "Семинар", 501, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := request.NewDetails(tt.purpose, tt.participants, true)
			if tt.wantErr {
				require.ErrorIs(t, err, request.ErrInvalidDetails)
				return
			}
			require.NoError(t, err)
			require.Equal(t, strings.TrimSpace(tt.purpose), d.Purpose)
		})
	}
}
