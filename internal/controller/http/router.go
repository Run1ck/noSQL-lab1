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

	mux.HandleFunc("POST /api/auth/register", v1.Register)
	mux.HandleFunc("POST /api/auth/login", v1.Login)
	mux.Handle("GET /api/auth/me", mw.Auth(http.HandlerFunc(v1.GetMe)))

	mux.Handle("POST /api/requests", mw.Auth(http.HandlerFunc(v1.CreateRequest)))
	mux.Handle("GET /api/requests", mw.Auth(http.HandlerFunc(v1.GetMyRequests)))
	mux.Handle("GET /api/requests/{id}", mw.Auth(http.HandlerFunc(v1.GetRequest)))
	mux.Handle("POST /api/requests/{id}/cancel", mw.Auth(http.HandlerFunc(v1.CancelRequest)))

	mux.Handle("GET /api/admin/requests", mw.Admin(http.HandlerFunc(v1.ListRequests)))
	mux.Handle("POST /api/admin/requests/{id}/approve", mw.Admin(http.HandlerFunc(v1.ApproveRequest)))
	mux.Handle("POST /api/admin/requests/{id}/reject", mw.Admin(http.HandlerFunc(v1.RejectRequest)))
	mux.Handle("GET /api/admin/services", mw.Admin(http.HandlerFunc(v1.GetAllServices)))
	mux.Handle("PUT /api/admin/services/{id}", mw.Admin(http.HandlerFunc(v1.SaveService)))
}
