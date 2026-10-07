package httpx

import (
	"booking/internal/auth"
	"net/http"
	"strings"
)

func NewMiddlewares(tokens auth.Tokens) Middlewares {
	authenticate := Authenticate(tokens)
	return Middlewares{
		Auth: authenticate,
		Admin: func(next http.Handler) http.Handler {
			return authenticate(RequireAdmin(next))
		},
	}
}

func Authenticate(tokens auth.Tokens) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				w.Header().Set("WWW-Authenticate", "Bearer")
				WriteError(w, auth.ErrUnauthorized)
				return
			}
			p, err := tokens.Parse(token)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
				WriteError(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.FromContext(r.Context())
		if !ok {
			WriteError(w, auth.ErrUnauthorized)
			return
		}
		if !p.IsAdmin() {
			WriteError(w, auth.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
