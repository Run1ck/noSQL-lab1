package usecase

import (
	"booking/internal/domain/service"
	"cmp"
	"time"
)

// Арифметика дней. У service.Date (общая зона) нет Compare и AddDays — они
// предложены в контракт; когда появятся, этот файл уйдёт.

func compareDates(a, b service.Date) int {
	return cmp.Or(cmp.Compare(a.Year, b.Year), cmp.Compare(a.Month, b.Month), cmp.Compare(a.Day, b.Day))
}

func daysBetween(from, to service.Date) int {
	return int(midnight(to).Sub(midnight(from)).Hours() / 24)
}

func addDays(d service.Date, n int) service.Date {
	return service.DateOf(midnight(d).AddDate(0, 0, n))
}

// midnight — начало дня в UTC: в UTC нет перевода часов, и в сутках всегда 24 часа.
func midnight(d service.Date) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}
