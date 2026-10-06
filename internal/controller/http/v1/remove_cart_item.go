package v1

import (
	"net/http"

	"booking/internal/auth"
	"booking/internal/dto"
	"booking/internal/httpx"
)

func (h *Handlers) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := auth.FromContext(ctx)
	if !ok {
		httpx.WriteError(w, auth.ErrUnauthorized)

		return
	}

	input := dto.RemoveCartItemInput{
		UserID:    p.UserID,
		ServiceID: r.PathValue("serviceID"),
		Date:      r.PathValue("date"),
	}

	output, err := h.usecase.RemoveCartItem(ctx, input)
	if err != nil {
		httpx.WriteError(w, err)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, output)
}
