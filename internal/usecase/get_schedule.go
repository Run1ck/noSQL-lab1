package usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"booking/internal/domain/service"
	"booking/internal/dto"
)

const maxScheduleDays = 31

func (u *UseCase) GetSchedule(ctx context.Context, input dto.GetScheduleInput) (dto.GetScheduleOutput, error) {
	var output dto.GetScheduleOutput

	from, err := service.ParseDate(input.From)
	if err != nil {
		return output, fmt.Errorf("service.ParseDate: %w", err)
	}

	to, err := service.ParseDate(input.To)
	if err != nil {
		return output, fmt.Errorf("service.ParseDate: %w", err)
	}

	days := daysBetween(from, to) + 1
	if days < 1 || days > maxScheduleDays {
		return output, service.ErrInvalidRange
	}

	output.Days = make([]dto.Day, 0, days)

	for i := range days {
		date := addDays(from, i)

		bookings, err := u.postgres.GetBookings(ctx, date)
		if err != nil {
			return output, fmt.Errorf("u.postgres.GetBookings: %w", err)
		}

		booked := make([]string, 0, len(bookings))
		for _, b := range bookings {
			booked = append(booked, b.ServiceID)
		}

		slices.Sort(booked)

		output.Days = append(output.Days, dto.Day{Date: date.String(), Booked: booked})
	}

	return output, nil
}

func daysBetween(from, to service.Date) int {
	return int(midnight(to).Sub(midnight(from)).Hours() / 24)
}

func addDays(d service.Date, n int) service.Date {
	return service.DateOf(midnight(d).AddDate(0, 0, n))
}

func midnight(d service.Date) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}
