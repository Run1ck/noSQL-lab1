package httpx

import (
	"booking/internal/auth"
	"booking/internal/domain/user"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

const testJWTSecret = "test-secret-at-least-32-bytes-long!"

func issue(t *testing.T, tokens auth.Tokens, role user.Role) (string, auth.Principal) {
	t.Helper()
	p := auth.Principal{UserID: uuid.New(), Role: role}
	token, _, err := tokens.Issue(p)
	if err != nil {
		t.Fatal(err)
	}
	return token, p
}

func serveAuth(mw func(http.Handler) http.Handler, header string) (*httptest.ResponseRecorder, *auth.Principal) {
	var got *auth.Principal
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := auth.FromContext(r.Context()); ok {
			got = &p
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/cart", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	return rec, got
}

func TestAuth_PassesValidToken(t *testing.T) {
	tokens := auth.NewJWT(testJWTSecret, time.Hour)
	mw := NewMiddlewares(tokens)
	token, want := issue(t, tokens, user.RoleUser)

	for _, header := range []string{"Bearer " + token, "bearer " + token} {
		rec, got := serveAuth(mw.Auth, header)
		if rec.Code != http.StatusOK || got == nil || *got != want {
			t.Errorf("%q: code=%d principal=%v, want 200 and %v", header, rec.Code, got, want)
		}
	}
}

func TestAuth_Rejects(t *testing.T) {
	tokens := auth.NewJWT(testJWTSecret, time.Hour)
	mw := NewMiddlewares(tokens)
	token, _ := issue(t, tokens, user.RoleAdmin)
	foreign, _ := issue(t, auth.NewJWT("another-secret-at-least-32-bytes!!", time.Hour), user.RoleAdmin)

	tests := []struct {
		name   string
		header string
		code   string
	}{
		{"no header", "", "unauthorized"},
		{"no scheme", token, "unauthorized"},
		{"basic", "Basic dXNlcjpwYXNz", "unauthorized"},
		{"empty token", "Bearer ", "unauthorized"},
		{"garbage", "Bearer not-a-token", "invalid_token"},
		{"wrong key", "Bearer " + foreign, "invalid_token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, got := serveAuth(mw.Auth, tt.header)
			if rec.Code != http.StatusUnauthorized || got != nil {
				t.Fatalf("code=%d reached=%v, want 401 not reached", rec.Code, got != nil)
			}
			if code := errorCode(t, rec); code != tt.code {
				t.Errorf("code %q, want %q", code, tt.code)
			}
			if rec.Header().Get("WWW-Authenticate") == "" {
				t.Error("missing WWW-Authenticate")
			}
		})
	}
}

func TestAdmin(t *testing.T) {
	tokens := auth.NewJWT(testJWTSecret, time.Hour)
	mw := NewMiddlewares(tokens)
	adminToken, admin := issue(t, tokens, user.RoleAdmin)
	userToken, _ := issue(t, tokens, user.RoleUser)

	rec, got := serveAuth(mw.Admin, "Bearer "+adminToken)
	if rec.Code != http.StatusOK || got == nil || *got != admin {
		t.Errorf("admin: code=%d principal=%v, want 200 and %v", rec.Code, got, admin)
	}

	rec, got = serveAuth(mw.Admin, "Bearer "+userToken)
	if rec.Code != http.StatusForbidden || got != nil || errorCode(t, rec) != "forbidden" {
		t.Errorf("user: code=%d reached=%v, want 403 forbidden", rec.Code, got != nil)
	}

	rec, got = serveAuth(mw.Admin, "")
	if rec.Code != http.StatusUnauthorized || got != nil || errorCode(t, rec) != "unauthorized" {
		t.Errorf("anonymous: code=%d reached=%v, want 401 unauthorized", rec.Code, got != nil)
	}
}

func TestRequireAdmin_WithoutPrincipal(t *testing.T) {
	rec, got := serveAuth(RequireAdmin, "")
	if rec.Code != http.StatusUnauthorized || got != nil || errorCode(t, rec) != "unauthorized" {
		t.Errorf("code=%d reached=%v, want 401 unauthorized", rec.Code, got != nil)
	}
}
