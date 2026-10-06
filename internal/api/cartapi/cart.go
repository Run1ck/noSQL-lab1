// Package cartapi — временная корзина пользователя: /api/cart, всё под Auth.
package cartapi

import (
	"booking/internal/auth"
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/httpx"
	"booking/internal/usecase"
	"math"
	"net/http"

	"github.com/google/uuid"
)

type Module struct {
	cart *usecase.Cart
}

func New(cart *usecase.Cart) *Module {
	return &Module{cart: cart}
}

func (m *Module) Register(mux *http.ServeMux, mw httpx.Middlewares) {
	mux.Handle("GET /api/cart", mw.Auth(withUser(m.get)))
	mux.Handle("POST /api/cart/items", mw.Auth(withUser(m.addItem)))
	mux.Handle("DELETE /api/cart/items/{serviceID}/{date}", mw.Auth(withUser(m.removeItem)))
	mux.Handle("DELETE /api/cart", mw.Auth(withUser(m.clear)))
}

type itemResponse struct {
	ServiceID string       `json:"service_id"`
	Date      service.Date `json:"date"`
}

type cartResponse struct {
	// Items — в порядке use case'а; у пустой корзины [], не null.
	Items []itemResponse `json:"items"`
	// ExpiresIn — секунд до истечения с округлением вверх: живая корзина не
	// покажет 0; 0 — корзины нет.
	ExpiresIn int `json:"expires_in"`
}

type addItemRequest struct {
	ServiceID string `json:"service_id"`
	// Date — строкой: с service.Date ошибка разбора пришла бы из DecodeJSON
	// как bad_request, а по контракту нужен invalid_date.
	Date string `json:"date"`
}

func (m *Module) get(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	view, err := m.cart.Get(r.Context(), userID)
	writeCart(w, view, err)
}

func (m *Module) addItem(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	var req addItemRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	date, err := service.ParseDate(req.Date)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	view, err := m.cart.AddItem(r.Context(), userID, cart.Item{ServiceID: req.ServiceID, Date: date})
	writeCart(w, view, err)
}

func (m *Module) removeItem(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	date, err := service.ParseDate(r.PathValue("date"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	view, err := m.cart.RemoveItem(r.Context(), userID, cart.Item{ServiceID: r.PathValue("serviceID"), Date: date})
	writeCart(w, view, err)
}

func (m *Module) clear(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	if err := m.cart.Clear(r.Context(), userID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeCart отвечает корзиной или ошибкой use case'а.
func writeCart(w http.ResponseWriter, view usecase.CartView, err error) {
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	items := make([]itemResponse, 0, len(view.Items))
	for _, it := range view.Items {
		items = append(items, itemResponse{ServiceID: it.ServiceID, Date: it.Date})
	}
	httpx.WriteJSON(w, http.StatusOK, cartResponse{
		Items:     items,
		ExpiresIn: int(math.Ceil(view.ExpiresIn.Seconds())),
	})
}

// withUser передаёт обработчику пользователя из контекста. За mw.Auth он есть
// всегда; проверка — на случай, если маршрут зарегистрируют без Auth: тогда
// 401, а не корзина нулевого UUID.
func withUser(h func(http.ResponseWriter, *http.Request, uuid.UUID)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.FromContext(r.Context())
		if !ok {
			httpx.WriteError(w, auth.ErrUnauthorized)
			return
		}
		h(w, r, p.UserID)
	})
}
