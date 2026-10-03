package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const maxBodyBytes = 1 << 20

// ErrBadRequest — некорректный запрос: битый JSON, лишние поля, неверный
// параметр пути. Обработчики оборачивают его: fmt.Errorf("%w: ...", ErrBadRequest).
var ErrBadRequest = errors.New("bad request")

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// DecodeJSON читает тело запроса в dst; неизвестные поля и тело больше
// 1 МБ — ErrBadRequest.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	return nil
}
