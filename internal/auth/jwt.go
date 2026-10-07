package auth

import (
	"booking/internal/domain/user"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWT struct {
	secret []byte
	ttl    time.Duration
}

var _ Tokens = (*JWT)(nil)

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWT(secret string, ttl time.Duration) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl}
}

func (j *JWT) Issue(p Principal) (string, time.Time, error) {
	now := time.Now()
	c := claims{
		Role: string(p.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, c.ExpiresAt.Time, nil
}

func (j *JWT) Parse(token string) (Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	role, err := user.ParseRole(c.Role)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	return Principal{UserID: id, Role: role}, nil
}
