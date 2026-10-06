package usecase

import (
	"booking/internal/domain/booking"
	"booking/internal/fake"
	"context"
	"errors"
	"slices"
	"testing"
)

func TestSchedule_Range(t *testing.T) {
	uc := NewSchedule(fake.NewBookings(
		booking.Booking{ServiceID: "room-101", Date: fake.Date("2026-10-05"), RequestID: 1},
		booking.Booking{ServiceID: "lab-1", Date: fake.Date("2026-10-05"), RequestID: 2},
		booking.Booking{ServiceID: "room-101", Date: fake.Date("2026-10-07"), RequestID: 3},
	))

	days, err := uc.Range(context.Background(), fake.Date("2026-10-05"), fake.Date("2026-10-07"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Day{
		{fake.Date("2026-10-05"), []string{"lab-1", "room-101"}}, // по возрастанию
		{fake.Date("2026-10-06"), []string{}},
		{fake.Date("2026-10-07"), []string{"room-101"}},
	}
	if !slices.EqualFunc(days, want, func(a, b Day) bool { return a.Date == b.Date && slices.Equal(a.Booked, b.Booked) }) {
		t.Fatalf("days:\n got %v\nwant %v", days, want)
	}
	// Фейк отдаёт nil за свободный день, а use case — пустой срез.
	if days[1].Booked == nil {
		t.Fatal("free day: Booked must be empty, not nil")
	}
}

func TestSchedule_Ranges(t *testing.T) {
	uc := NewSchedule(fake.NewBookings())

	for _, tc := range []struct {
		from, to string
		want     []string // все дни, если коротко, иначе первый и последний
		days     int
	}{
		{"2026-10-05", "2026-10-05", []string{"2026-10-05"}, 1},
		{"2028-02-28", "2028-03-01", []string{"2028-02-28", "2028-02-29", "2028-03-01"}, 3}, // високосный год
		{"2026-12-31", "2027-01-01", []string{"2026-12-31", "2027-01-01"}, 2},
		{"2026-03-28", "2026-03-30", []string{"2026-03-28", "2026-03-29", "2026-03-30"}, 3}, // перевод часов в Европе
		{"2026-10-01", "2026-10-31", []string{"2026-10-01", "2026-10-31"}, MaxScheduleDays},
	} {
		days, err := uc.Range(context.Background(), fake.Date(tc.from), fake.Date(tc.to))
		if err != nil {
			t.Errorf("%s..%s: %v", tc.from, tc.to, err)
			continue
		}
		var got []string
		for _, d := range days {
			got = append(got, d.Date.String())
		}
		if len(got) != tc.days {
			t.Errorf("%s..%s: %d days, want %d", tc.from, tc.to, len(got), tc.days)
			continue
		}
		if len(tc.want) < tc.days { // проверяем только края
			got = []string{got[0], got[len(got)-1]}
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s..%s: got %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestSchedule_InvalidRange(t *testing.T) {
	uc := NewSchedule(fake.NewBookings())

	for _, tc := range [][2]string{
		{"2026-10-07", "2026-10-05"}, // from позже to
		{"2026-10-01", "2026-11-01"}, // 32 дня
	} {
		if _, err := uc.Range(context.Background(), fake.Date(tc[0]), fake.Date(tc[1])); !errors.Is(err, ErrInvalidRange) {
			t.Errorf("%s..%s: err = %v, want ErrInvalidRange", tc[0], tc[1], err)
		}
	}
}

func TestSchedule_RepositoryError(t *testing.T) {
	bookings := fake.NewBookings()
	bookings.Err = errors.New("db down")

	_, err := NewSchedule(bookings).Range(context.Background(), fake.Date("2026-10-05"), fake.Date("2026-10-05"))
	if !errors.Is(err, bookings.Err) {
		t.Fatalf("err = %v, want repository error", err)
	}
}
