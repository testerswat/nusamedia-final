package middleware

import (
	"context"
	"github.com/nusamedia/platform/apps/api/internal/auth"
	"github.com/nusamedia/platform/apps/api/internal/httpx"
	"net/http"
	"strings"
)

type key int

const claimsKey key = 1

func Require(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			httpx.Error(w, 401, "UNAUTHENTICATED", "login required")
			return
		}
		c, e := auth.Parse(secret, strings.TrimPrefix(h, "Bearer "))
		if e != nil {
			httpx.Error(w, 401, "INVALID_TOKEN", "invalid token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, c)))
	})
}
func Claims(r *http.Request) auth.Claims {
	c, _ := r.Context().Value(claimsKey).(auth.Claims)
	return c
}
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := Claims(r)
		if c.Role != "ADMIN_ROOT" && c.Role != "ADMIN" {
			httpx.Error(w, 403, "FORBIDDEN", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
