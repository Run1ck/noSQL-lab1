package httpx

import (
	"booking/internal/auth"
	"booking/internal/domain/cart"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/domain/user"
	"booking/internal/ratelimit"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
)

// ErrorResponse — тело любого ответа с ошибкой:
//
//	{"error": {"code": "slot_booked", "message": "slot is already booked"}}
//
// По code клиент (UI, нагрузочный тест) решает, что делать; message — для человека.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// errorMap — единственное место, где ошибки превращаются в HTTP-статус и код.
// Кто заводит новую ошибку, тот и добавляет её сюда.
var errorMap = []struct {
	err    error
	status int
	code   string
}{
	{ErrBadRequest, http.StatusBadRequest, "bad_request"},

	{auth.ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
	{auth.ErrInvalidToken, http.StatusUnauthorized, "invalid_token"},
	{auth.ErrForbidden, http.StatusForbidden, "forbidden"},

	{user.ErrInvalidCredentials, http.StatusUnauthorized, "invalid_credentials"},
	{user.ErrInvalidPassword, http.StatusBadRequest, "invalid_password"},
	{user.ErrInvalidRole, http.StatusBadRequest, "invalid_role"},
	{user.ErrLoginTaken, http.StatusConflict, "login_taken"},
	{user.ErrNotFound, http.StatusNotFound, "user_not_found"},

	{service.ErrInvalid, http.StatusBadRequest, "invalid_service"},
	{service.ErrInvalidKind, http.StatusBadRequest, "invalid_kind"},
	{service.ErrInvalidDate, http.StatusBadRequest, "invalid_date"},
	{service.ErrNotFound, http.StatusNotFound, "service_not_found"},

	{cart.ErrInvalidUUID, http.StatusBadRequest, "invalid_id"},
	{cart.ErrItemNotFound, http.StatusNotFound, "item_not_found"},
	{cart.ErrPastDate, http.StatusUnprocessableEntity, "past_date"},
	{cart.ErrServiceUnavailable, http.StatusUnprocessableEntity, "service_unavailable"},
	{cart.ErrSlotBooked, http.StatusConflict, "slot_booked"},

	{request.ErrEmptyCart, http.StatusUnprocessableEntity, "empty_cart"},
	{request.ErrInvalidStatus, http.StatusBadRequest, "invalid_status"},
	{request.ErrNotFound, http.StatusNotFound, "request_not_found"},
	{request.ErrNotOwner, http.StatusForbidden, "forbidden"},
	{request.ErrAlreadyProcessed, http.StatusConflict, "already_processed"},
	{request.ErrSlotBooked, http.StatusConflict, "slot_booked"},
}

// WriteError пишет ответ с ошибкой. На ошибку не из errorMap отвечает 500
// без подробностей, а саму ошибку пишет в лог.
func WriteError(w http.ResponseWriter, err error) {
	var limited *ratelimit.LimitedError
	if errors.As(err, &limited) {
		secs := max(1, int(math.Ceil(limited.RetryAfter.Seconds())))
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		code := "rate_limited"
		if limited.Banned {
			code = "banned"
		}
		writeError(w, http.StatusTooManyRequests, code, err.Error())
		return
	}
	for _, m := range errorMap {
		if errors.Is(err, m.err) {
			writeError(w, m.status, m.code, err.Error())
			return
		}
	}
	slog.Error("unhandled error", "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "internal server error")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}
