package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"booking/pkg/httpx"
)

func requestID(r *http.Request) (int64, error) {
	s := r.PathValue("id")

	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: invalid request id %q", httpx.ErrBadRequest, s)
	}

	return id, nil
}
