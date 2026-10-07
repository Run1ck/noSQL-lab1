package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"booking/internal/auth"
	controllerhttp "booking/internal/controller/http"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/domain/user"
	"booking/internal/usecase"
	"booking/internal/usecase/mocks"
	"booking/pkg/httpx"
)

const Any = mock.Anything

type env struct {
	server   *httptest.Server
	postgres *mocks.Postgres
	tokens   auth.Tokens
}

func newEnv(t *testing.T) *env {
	t.Helper()

	postgres := new(mocks.Postgres)
	tokens := auth.NewJWT("test-secret-at-least-32-bytes-long!", time.Hour)
	mux := http.NewServeMux()
	controllerhttp.Router(mux, usecase.New(postgres, new(mocks.Redis), tokens, nil), httpx.NewMiddlewares(tokens))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &env{server: server, postgres: postgres, tokens: tokens}
}

func (e *env) token(t *testing.T, role user.Role) (string, uuid.UUID) {
	t.Helper()

	id := uuid.New()
	token, _, err := e.tokens.Issue(auth.Principal{UserID: id, Role: role})
	require.NoError(t, err)

	return token, id
}

func (e *env) do(t *testing.T, method, path, token, body string) (int, map[string]any) {
	t.Helper()

	req, err := http.NewRequest(method, e.server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out map[string]any
	if len(raw) > 0 && raw[0] == '{' {
		require.NoError(t, json.Unmarshal(raw, &out))
	}

	return resp.StatusCode, out
}

func errorCode(out map[string]any) string {
	e, _ := out["error"].(map[string]any)
	code, _ := e["code"].(string)

	return code
}

func Test_Register(t *testing.T) {
	e := newEnv(t)
	e.postgres.On("CreateUser", Any, Any).Return(nil).Once()
	e.postgres.On("CreateUser", Any, Any).Return(user.ErrLoginTaken).Once()

	code, out := e.do(t, http.MethodPost, "/api/auth/register", "", `{"login":"ivanov","password":"password123"}`)
	require.Equal(t, http.StatusCreated, code)
	require.Equal(t, "ivanov", out["login"])
	require.Equal(t, "user", out["role"])

	code, out = e.do(t, http.MethodPost, "/api/auth/register", "", `{"login":"ivanov","password":"password123"}`)
	require.Equal(t, http.StatusConflict, code)
	require.Equal(t, "login_taken", errorCode(out))

	code, out = e.do(t, http.MethodPost, "/api/auth/register", "", `{"login":"ivanov","password":"password123","role":"admin"}`)
	require.Equal(t, http.StatusBadRequest, code)
	require.Equal(t, "bad_request", errorCode(out))
}

func Test_LoginAndMe(t *testing.T) {
	e := newEnv(t)
	usr, err := user.New("ivanov", "password123", user.RoleUser)
	require.NoError(t, err)
	e.postgres.On("GetUserByLogin", Any, "ivanov").Return(usr, nil)
	e.postgres.On("GetUserByID", Any, usr.ID).Return(usr, nil)

	code, out := e.do(t, http.MethodPost, "/api/auth/login", "", `{"login":"ivanov","password":"wrong-password"}`)
	require.Equal(t, http.StatusUnauthorized, code)
	require.Equal(t, "invalid_credentials", errorCode(out))

	code, out = e.do(t, http.MethodPost, "/api/auth/login", "", `{"login":"ivanov","password":"password123"}`)
	require.Equal(t, http.StatusOK, code)
	token, _ := out["token"].(string)
	require.NotEmpty(t, token)
	require.NotEmpty(t, out["expires_at"])

	code, out = e.do(t, http.MethodGet, "/api/auth/me", token, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, usr.ID.String(), out["id"])

	code, out = e.do(t, http.MethodGet, "/api/auth/me", "", "")
	require.Equal(t, http.StatusUnauthorized, code)
	require.Equal(t, "unauthorized", errorCode(out))
}

func Test_Requests(t *testing.T) {
	e := newEnv(t)
	token, userID := e.token(t, user.RoleUser)
	e.postgres.On("GetRequest", Any, int64(42)).Return(&request.Request{ID: 42, UserID: userID, Status: request.StatusNew}, nil)
	e.postgres.On("GetRequest", Any, int64(7)).Return(&request.Request{ID: 7, UserID: uuid.New(), Status: request.StatusNew}, nil)
	e.postgres.On("Cancel", Any, Any).Return(nil)

	code, out := e.do(t, http.MethodGet, "/api/requests/42", token, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, float64(42), out["id"])
	require.Nil(t, out["processed_at"])

	code, out = e.do(t, http.MethodGet, "/api/requests/7", token, "")
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, "forbidden", errorCode(out))

	code, out = e.do(t, http.MethodGet, "/api/requests/abc", token, "")
	require.Equal(t, http.StatusBadRequest, code)
	require.Equal(t, "bad_request", errorCode(out))

	code, out = e.do(t, http.MethodPost, "/api/requests/42/cancel", token, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "cancelled", out["status"])
}

func Test_Admin(t *testing.T) {
	e := newEnv(t)
	adminToken, adminID := e.token(t, user.RoleAdmin)
	userToken, _ := e.token(t, user.RoleUser)
	for range 2 {
		e.postgres.On("GetRequest", Any, int64(42)).Return(&request.Request{ID: 42, UserID: uuid.New(), Status: request.StatusNew}, nil).Once()
	}
	e.postgres.On("Approve", Any, Any).Return(request.ErrSlotBooked).Once()
	e.postgres.On("Approve", Any, Any).Return(nil).Once()
	e.postgres.On("GetRequestsByStatus", Any, request.StatusNew).Return([]*request.Request{}, nil)
	e.postgres.On("SaveService", Any, Any).Return(nil)

	code, out := e.do(t, http.MethodGet, "/api/admin/requests", userToken, "")
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, "forbidden", errorCode(out))

	code, out = e.do(t, http.MethodGet, "/api/admin/requests?status=done", adminToken, "")
	require.Equal(t, http.StatusBadRequest, code)
	require.Equal(t, "invalid_status", errorCode(out))

	code, _ = e.do(t, http.MethodGet, "/api/admin/requests?status=new", adminToken, "")
	require.Equal(t, http.StatusOK, code)

	code, out = e.do(t, http.MethodPost, "/api/admin/requests/42/approve", adminToken, `{"comment":"ok"}`)
	require.Equal(t, http.StatusConflict, code)
	require.Equal(t, "slot_booked", errorCode(out))

	code, out = e.do(t, http.MethodPost, "/api/admin/requests/42/approve", adminToken, `{"comment":"ok"}`)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "approved", out["status"])
	require.Equal(t, adminID.String(), out["processed_by"])

	code, out = e.do(t, http.MethodPut, "/api/admin/services/room-101", adminToken, `{"name":"Аудитория 101","kind":"room","active":true}`)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "room-101", out["id"])
	e.postgres.AssertCalled(t, "SaveService", Any, &service.Service{ID: "room-101", Name: "Аудитория 101", Kind: service.Room, Active: true})

	code, out = e.do(t, http.MethodPut, "/api/admin/services/room-101", adminToken, `{"name":"x","kind":"garage","active":true}`)
	require.Equal(t, http.StatusBadRequest, code)
	require.Equal(t, "invalid_kind", errorCode(out))
}
