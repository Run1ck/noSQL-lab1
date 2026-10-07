package v1

import (
	"net/http"

	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) SaveService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input := dto.SaveServiceInput{}

	err := httpx.DecodeJSON(w, r, &input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	input.ID = r.PathValue("id")

	output, err := h.usecase.SaveService(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}
