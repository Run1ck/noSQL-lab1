package v1

import (
	"net/http"

	"booking/internal/auth"
	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) ClearCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := auth.FromContext(ctx)
	if !ok {
		httpx.WriteError(w, auth.ErrUnauthorized)

		return
	}

	input := dto.ClearCartInput{
		UserID: p.UserID,
	}

	err := h.usecase.ClearCart(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
