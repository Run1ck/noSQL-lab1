// Package scheduleapi — расписание занятости: GET /api/schedule, под Auth.
package scheduleapi

import (
	"booking/internal/domain/service"
	"booking/internal/httpx"
	"booking/internal/usecase"
	"errors"
	"fmt"
	"net/http"
)

type Module struct {
	schedule *usecase.Schedule
}

func New(schedule *usecase.Schedule) *Module {
	return &Module{schedule: schedule}
}

func (m *Module) Register(mux *http.ServeMux, mw httpx.Middlewares) {
	mux.Handle("GET /api/schedule", mw.Auth(http.HandlerFunc(m.get)))
}

type dayResponse struct {
	Date service.Date `json:"date"`
	// Booked — ID занятых услуг по возрастанию; у свободного дня [], не null.
	Booked []string `json:"booked"`
}

type scheduleResponse struct {
	Days []dayResponse `json:"days"`
}

// get отдаёт каждый день диапазона [from, to] включительно.
func (m *Module) get(w http.ResponseWriter, r *http.Request) {
	from, err := service.ParseDate(r.URL.Query().Get("from"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	to, err := service.ParseDate(r.URL.Query().Get("to"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	days, err := m.schedule.Range(r.Context(), from, to)
	if errors.Is(err, usecase.ErrInvalidRange) {
		// Ошибки use case'ов в errorMap нет (общая зона) — переводим сами.
		err = fmt.Errorf("%w: %v", httpx.ErrBadRequest, err)
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	out := scheduleResponse{Days: make([]dayResponse, 0, len(days))}
	for _, d := range days {
		out.Days = append(out.Days, dayResponse{Date: d.Date, Booked: d.Booked})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
