package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/nusamedia/platform/apps/api/internal/httpx"
)

// RequireAdminDB revalidates the user's current database role on every request.
// This prevents a revoked/stale JWT from retaining administrative access.
func RequireAdminDB(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := Claims(r)
		var role string
		if err := db.QueryRowContext(r.Context(), `SELECT role FROM users WHERE id=$1`, c.UserID).Scan(&role); err != nil {
			httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "admin identity is no longer active")
			return
		}
		switch strings.ToUpper(role) {
		case "ADMIN_ROOT", "ADMIN_LEVEL_2", "ADMIN", "DEVELOPER", "RELEASE_MANAGER", "MODERATOR", "FINANCE", "MANAGER", "SECURITY_ADMIN", "STAKEHOLDER", "FINANCE_MANAGER", "FINANCE_REVIEWER", "VERIFICATION_MANAGER", "VERIFICATION_REVIEWER", "OPERATIONS_MANAGER":
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), roleKey{}, role)))
		default:
			httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		}
	})
}

type roleKey struct{}

func CurrentRole(r *http.Request) string { v, _ := r.Context().Value(roleKey{}).(string); return v }

func RequireRootDB(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := Claims(r)
		var role string
		if err := db.QueryRowContext(r.Context(), `SELECT role FROM users WHERE id=$1`, c.UserID).Scan(&role); err != nil || strings.ToUpper(role) != "ADMIN_ROOT" {
			httpx.Error(w, http.StatusForbidden, "ROOT_REQUIRED", "Admin Root approval is required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), roleKey{}, role)))
	})
}
