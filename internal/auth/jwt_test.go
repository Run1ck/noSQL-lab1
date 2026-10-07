package auth

import (
	"booking/internal/domain/user"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-at-least-32-bytes-long!"

func signClaims(t *testing.T, method jwt.SigningMethod, key any, c claims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, c).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func validClaims(sub, role string) claims {
	now := time.Now()
	return claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}

func TestJWT_IssueParse(t *testing.T) {
	j := NewJWT(testSecret, time.Hour)
	want := Principal{UserID: uuid.New(), Role: user.RoleAdmin}

	token, expiresAt, err := j.Issue(want)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(expiresAt); d < 59*time.Minute || d > time.Hour {
		t.Errorf("expiresAt in %s, want about 1h", d)
	}

	got, err := j.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestJWT_ParseRejects(t *testing.T) {
	j := NewJWT(testSecret, time.Hour)
	id := uuid.New().String()

	userToken, _, err := j.Issue(Principal{UserID: uuid.New(), Role: user.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	adminToken, _, err := j.Issue(Principal{UserID: uuid.New(), Role: user.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	u, a := strings.Split(userToken, "."), strings.Split(adminToken, ".")
	tampered := strings.Join([]string{u[0], a[1], u[2]}, ".")

	expired, _, err := NewJWT(testSecret, -time.Minute).Issue(Principal{UserID: uuid.New(), Role: user.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	foreign, _, err := NewJWT("another-secret-at-least-32-bytes!!", time.Hour).Issue(Principal{UserID: uuid.New(), Role: user.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	noExp := validClaims(id, "user")
	noExp.ExpiresAt = nil

	tests := map[string]string{
		"garbage":     "not-a-token",
		"empty":       "",
		"tampered":    tampered,
		"expired":     expired,
		"wrong key":   foreign,
		"alg none":    signClaims(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims(id, "admin")),
		"hs512":       signClaims(t, jwt.SigningMethodHS512, []byte(testSecret), validClaims(id, "admin")),
		"no exp":      signClaims(t, jwt.SigningMethodHS256, []byte(testSecret), noExp),
		"bad subject": signClaims(t, jwt.SigningMethodHS256, []byte(testSecret), validClaims("not-a-uuid", "user")),
		"bad role":    signClaims(t, jwt.SigningMethodHS256, []byte(testSecret), validClaims(id, "root")),
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := j.Parse(token); !errors.Is(err, ErrInvalidToken) {
				t.Errorf("got %v, want ErrInvalidToken", err)
			}
		})
	}
}
