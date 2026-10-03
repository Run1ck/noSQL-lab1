package httpx

import "net/http"

// Module — HTTP-модуль одной части API. Модуль сам регистрирует свои
// маршруты, а main только перечисляет модули, поэтому новый эндпоинт
// не требует правок в общих файлах.
type Module interface {
	Register(mux *http.ServeMux, mw Middlewares)
}

// Middlewares — обёртки доступа, которые main передаёт модулям.
type Middlewares struct {
	// Auth пропускает запрос с валидным JWT и кладёт в контекст
	// auth.Principal; иначе 401.
	Auth func(http.Handler) http.Handler
	// Admin — то же, что Auth, плюс 403 для не-администратора.
	Admin func(http.Handler) http.Handler
}
