package v1

import (
	"net/http"

	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) ListRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input := dto.ListRequestsInput{
		Status: r.URL.Query().Get("status"),
	}

	output, err := h.usecase.ListRequests(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output.Requests)
}
