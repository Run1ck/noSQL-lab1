package v1

import (
	"net/http"

	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input := dto.LoginInput{}

	err := httpx.DecodeJSON(w, r, &input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	output, err := h.usecase.Login(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}
