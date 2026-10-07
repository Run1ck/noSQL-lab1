package http

import (
	"net/http"

	ver1 "booking/internal/controller/http/v1"
	"booking/internal/usecase"
	"booking/pkg/httpx"
)

func Router(mux *http.ServeMux, uc *usecase.UseCase, mw httpx.Middlewares) {
	v1 := ver1.New(uc)

	mux.HandleFunc("GET /api/services", v1.GetServices)
	mux.Handle("GET /api/schedule", mw.Auth(http.HandlerFunc(v1.GetSchedule)))

	mux.Handle("GET /api/cart", mw.Auth(http.HandlerFunc(v1.GetCart)))
	mux.Handle("POST /api/cart/items", mw.Auth(http.HandlerFunc(v1.AddCartItem)))
	mux.Handle("DELETE /api/cart/items/{serviceID}/{date}", mw.Auth(http.HandlerFunc(v1.RemoveCartItem)))
	mux.Handle("DELETE /api/cart", mw.Auth(http.HandlerFunc(v1.ClearCart)))
}
