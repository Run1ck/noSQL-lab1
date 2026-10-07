package v1

import (
	"net/http"

	"booking/internal/auth"
	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) CancelRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := auth.FromContext(ctx)
	if !ok {
		httpx.WriteError(w, auth.ErrUnauthorized)

		return
	}

	id, err := requestID(r)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	input := dto.CancelRequestInput{
		UserID: p.UserID,
		ID:     id,
	}

	output, err := h.usecase.CancelRequest(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}
