package v1

import (
	"net/http"

	"booking/internal/auth"
	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) CreateRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := auth.FromContext(ctx)
	if !ok {
		httpx.WriteError(w, auth.ErrUnauthorized)

		return
	}

	input := dto.CreateRequestInput{
		UserID: p.UserID,
	}

	output, err := h.usecase.CreateRequest(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, output)
}
