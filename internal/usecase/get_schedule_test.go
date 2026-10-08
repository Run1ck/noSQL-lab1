package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"booking/internal/domain/booking"
	"booking/internal/domain/service"
	"booking/internal/dto"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
)

func date(s string) service.Date {
	d, err := service.ParseDate(s)
	if err != nil {
		panic(err)
	}

	return d
}

func Test_GetSchedule_Success(t *testing.T) {
	postgres := new(mocks.Postgres)
	postgres.On("GetBookings", Any, date("2026-10-05")).Return([]booking.Booking{
		{ServiceID: "room-101", Date: date("2026-10-05"), RequestID: 1},
		{ServiceID: "lab-1", Date: date("2026-10-05"), RequestID: 2},
	}, nil)
	postgres.On("GetBookings", Any, date("2026-10-06")).Return(nil, nil)

	u := usecase.New(postgres, nil, nil, nil)

	{
		input := dto.GetScheduleInput{From: "2026-10-05", To: "2026-10-06"}
		output := dto.GetScheduleOutput{Days: []dto.Day{
			{Date: "2026-10-05", Booked: []string{"lab-1", "room-101"}},
			{Date: "2026-10-06", Booked: []string{}},
		}}

		actual, err := u.GetSchedule(context.Background(), input)
		require.NoError(t, err)
		require.Equal(t, output, actual)
	}
}

func Test_GetSchedule_Ranges(t *testing.T) {
	postgres := new(mocks.Postgres)
	postgres.On("GetBookings", Any, Any).Return(nil, nil)

	u := usecase.New(postgres, nil, nil, nil)

	for _, tc := range []struct {
		from, to    string
		days        int
		first, last string
	}{
		{"2026-10-05", "2026-10-05", 1, "2026-10-05", "2026-10-05"},
		{"2026-10-01", "2026-10-31", 31, "2026-10-01", "2026-10-31"},
		{"2028-02-28", "2028-03-01", 3, "2028-02-28", "2028-03-01"},
		{"2026-03-28", "2026-03-30", 3, "2026-03-28", "2026-03-30"},
	} {
		actual, err := u.GetSchedule(context.Background(), dto.GetScheduleInput{From: tc.from, To: tc.to})
		require.NoError(t, err)
		require.Len(t, actual.Days, tc.days)
		require.Equal(t, tc.first, actual.Days[0].Date)
		require.Equal(t, tc.last, actual.Days[len(actual.Days)-1].Date)
	}
}

func Test_GetSchedule_InvalidInput(t *testing.T) {
	u := usecase.New(nil, nil, nil, nil)

	for _, tc := range []struct {
		from, to string
		err      error
	}{
		{"2026-02-30", "2026-03-01", service.ErrInvalidDate},
		{"2026-10-05", "", service.ErrInvalidDate},
		{"2026-10-07", "2026-10-05", service.ErrInvalidRange},
		{"2026-10-01", "2026-11-01", service.ErrInvalidRange},
	} {
		actual, err := u.GetSchedule(context.Background(), dto.GetScheduleInput{From: tc.from, To: tc.to})
		require.ErrorIs(t, err, tc.err)
		require.Equal(t, dto.GetScheduleOutput{}, actual)
	}
}

func Test_GetSchedule_Error(t *testing.T) {
	errDB := errors.New("db down")

	postgres := new(mocks.Postgres)
	postgres.On("GetBookings", Any, Any).Return(nil, errDB)

	u := usecase.New(postgres, nil, nil, nil)

	{
		_, err := u.GetSchedule(context.Background(), dto.GetScheduleInput{From: "2026-10-05", To: "2026-10-05"})
		require.ErrorIs(t, err, errDB)
	}
}
