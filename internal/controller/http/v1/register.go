package v1

import (
	"net/http"

	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input := dto.RegisterInput{}

	err := httpx.DecodeJSON(w, r, &input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	output, err := h.usecase.Register(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, output)
}
