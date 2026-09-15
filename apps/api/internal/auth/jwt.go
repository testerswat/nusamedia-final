package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func Sign(secret, userID, role string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: userID, Role: role, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}}).SignedString([]byte(secret))
}
func Parse(secret, token string) (Claims, error) {
	var c Claims
	t, e := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) { return []byte(secret), nil })
	if e != nil || !t.Valid {
		return c, e
	}
	return c, nil
}
