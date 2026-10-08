package httpx

import "net/http"

// Middlewares — обёртки доступа, которые router.go вешает на маршруты.
type Middlewares struct {
	// Auth пропускает запрос с валидным JWT и кладёт в контекст
	// auth.Principal; иначе 401.
	Auth func(http.Handler) http.Handler
	// Admin — то же, что Auth, плюс 403 для не-администратора.
	Admin func(http.Handler) http.Handler
}
