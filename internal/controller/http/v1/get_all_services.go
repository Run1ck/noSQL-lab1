package v1

import (
	"net/http"

	"booking/pkg/httpx"
)

func (h *Handlers) GetAllServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	output, err := h.usecase.GetAllServices(ctx)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output.Services)
}
