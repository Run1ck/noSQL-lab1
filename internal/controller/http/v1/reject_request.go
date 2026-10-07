package v1

import (
	"net/http"

	"booking/pkg/httpx"
)

func (h *Handlers) RejectRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input, ok := processRequestInput(w, r)
	if !ok {
		return
	}

	output, err := h.usecase.RejectRequest(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}
