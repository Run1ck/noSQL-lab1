package v1

import (
	"net/http"

	"booking/internal/auth"
	"booking/internal/dto"
	"booking/pkg/httpx"
)

func (h *Handlers) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input, ok := processRequestInput(w, r)
	if !ok {
		return
	}

	output, err := h.usecase.ApproveRequest(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}

func processRequestInput(w http.ResponseWriter, r *http.Request) (dto.ProcessRequestInput, bool) {
	input := dto.ProcessRequestInput{}

	p, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, auth.ErrUnauthorized)

		return input, false
	}

	id, err := requestID(r)
	if err != nil {
		httpx.WriteError(w, err)

		return input, false
	}

	err = httpx.DecodeJSON(w, r, &input)
	if err != nil {
		httpx.WriteError(w, err)

		return input, false
	}

	input.AdminID = p.UserID
	input.ID = id

	return input, true
}
