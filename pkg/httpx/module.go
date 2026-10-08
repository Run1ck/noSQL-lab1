package httpx

import "net/http"

type Module interface {
	Register(mux *http.ServeMux, mw Middlewares)
}

type Middlewares struct {
	Auth  func(http.Handler) http.Handler
	Admin func(http.Handler) http.Handler
}
