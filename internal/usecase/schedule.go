package usecase

import (
	"booking/internal/domain/booking"
	"booking/internal/domain/service"
	"context"
	"errors"
	"fmt"
	"slices"
)

// MaxScheduleDays — сколько дней расписания можно запросить за раз: каждый
// день — отдельный ListByDate (с кэшем — GET в Redis).
const MaxScheduleDays = 31

// ErrInvalidRange — from позже to или диапазон длиннее MaxScheduleDays.
var ErrInvalidRange = errors.New("invalid date range")

// Day — занятость одного дня: ID занятых услуг по возрастанию; у свободного
// дня — пустой срез.
type Day struct {
	Date   service.Date
	Booked []string
}

// Schedule — расписание занятости услуг.
type Schedule struct {
	bookings booking.Repository
}

// NewSchedule принимает booking.Repository; в main это кэш-декоратор поверх Postgres.
func NewSchedule(bookings booking.Repository) *Schedule {
	return &Schedule{bookings: bookings}
}

// Range — занятость каждого дня от from до to включительно.
func (uc *Schedule) Range(ctx context.Context, from, to service.Date) ([]Day, error) {
	n := daysBetween(from, to) + 1
	if n < 1 || n > MaxScheduleDays {
		return nil, fmt.Errorf("%w: %s..%s, from must not be after to, at most %d days",
			ErrInvalidRange, from, to, MaxScheduleDays)
	}

	days := make([]Day, 0, n)
	for i := range n {
		d := addDays(from, i)
		bookings, err := uc.bookings.ListByDate(ctx, d)
		if err != nil {
			return nil, err
		}
		booked := make([]string, 0, len(bookings))
		for _, b := range bookings {
			booked = append(booked, b.ServiceID)
		}
		slices.Sort(booked)
		days = append(days, Day{Date: d, Booked: booked})
	}
	return days, nil
}
