package middleware

import (
	"github.com/nusamedia/platform/apps/api/internal/httpx"
	"net/http"
)

// RequireRoot is deliberately stricter than RequireAdmin. Only ADMIN_ROOT can
// change infrastructure, integrations, roles, releases, or platform settings.
func RequireRoot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if Claims(r).Role != "ADMIN_ROOT" {
			httpx.Error(w, http.StatusForbidden, "ROOT_REQUIRED", "Admin Root approval is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
