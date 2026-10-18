package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxBodyBytes = 1 << 20

var ErrBadRequest = errors.New("bad request")

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	return nil
}

func DecodeOptionalJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	err := DecodeJSON(w, r, dst)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}
