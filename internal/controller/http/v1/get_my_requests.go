package v1

import (
	"net/http"

	"booking/internal/auth"
	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) GetMyRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := auth.FromContext(ctx)
	if !ok {
		httpx.WriteError(w, auth.ErrUnauthorized)

		return
	}

	input := dto.GetMyRequestsInput{
		UserID: p.UserID,
	}

	output, err := h.usecase.GetMyRequests(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output.Requests)
}
