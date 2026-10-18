package httpx

import "net/http"

type Middlewares struct {
	Auth  func(http.Handler) http.Handler
	Admin func(http.Handler) http.Handler
}
