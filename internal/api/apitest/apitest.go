// Package apitest — заглушки авторизации и хелперы HTTP-запросов для тестов
// HTTP-модулей; фейки репозиториев — в internal/fake.
package apitest

import (
	"booking/internal/auth"
	"booking/internal/domain/user"
	"booking/internal/httpx"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
)

// Тестовые пользователи.
var (
	Alice = auth.Principal{UserID: uuid.MustParse("8f1c2a4e-0b7d-4c3e-9a51-6d2f0e8b7c11"), Role: user.RoleUser}
	Bob   = auth.Principal{UserID: uuid.MustParse("3b9e7d10-5c2a-4f8e-b6d4-1a0c9e2f7b33"), Role: user.RoleUser}
)

// Middlewares — заглушка авторизации: пропускает всех и кладёт в контекст p,
// как настоящий middleware после проверки JWT.
func Middlewares(p auth.Principal) httpx.Middlewares {
	withPrincipal := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
		})
	}
	return httpx.Middlewares{Auth: withPrincipal, Admin: withPrincipal}
}

// RejectAll — заглушка авторизации, которая отвечает 401 на всё: так тест
// видит, какие маршруты закрыты Auth, а какие открыты всем.
func RejectAll() httpx.Middlewares {
	reject := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpx.WriteError(w, auth.ErrUnauthorized)
		})
	}
	return httpx.Middlewares{Auth: reject, Admin: reject}
}

// Mux регистрирует модуль в новом ServeMux, как это делает main.
func Mux(m httpx.Module, mw httpx.Middlewares) *http.ServeMux {
	mux := http.NewServeMux()
	m.Register(mux, mw)
	return mux
}

// Do выполняет запрос к h; пустой body — запрос без тела.
func Do(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, rd))
	return rec
}

// ErrorCode — code из тела ошибки {"error": {"code": …}}; "" — тело не ошибка.
func ErrorCode(rec *httptest.ResponseRecorder) string {
	var body httpx.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Error.Code
}
