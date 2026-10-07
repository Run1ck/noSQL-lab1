package v1

import (
	"net/http"

	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) GetSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input := dto.GetScheduleInput{
		From: r.URL.Query().Get("from"),
		To:   r.URL.Query().Get("to"),
	}

	output, err := h.usecase.GetSchedule(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}
