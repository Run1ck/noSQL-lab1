package v1

import (
	"net/http"

	"booking/internal/auth"
	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) AddCartItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := auth.FromContext(ctx)
	if !ok {
		httpx.WriteError(w, auth.ErrUnauthorized)

		return
	}

	input := dto.AddCartItemInput{}

	err := httpx.DecodeJSON(w, r, &input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	input.UserID = p.UserID

	output, err := h.usecase.AddCartItem(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}
