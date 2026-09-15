package httpapi

import (
	"database/sql"
	"github.com/nusamedia/platform/apps/api/internal/httpx"
	"net/http"
	"os"
	"strings"
)

func (a *Router) setupBootstrap(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-NusaMedia-Setup-Token")) == "" || r.Header.Get("X-NusaMedia-Setup-Token") != os.Getenv("NUSAMEDIA_SETUP_TOKEN") {
		httpx.Error(w, http.StatusForbidden, "SETUP_TOKEN_INVALID", "invalid setup token")
		return
	}
	var x struct{ Email, Password, DisplayName string }
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	x.Email = strings.TrimSpace(strings.ToLower(x.Email))
	x.DisplayName = strings.TrimSpace(x.DisplayName)
	if x.Email == "" || x.DisplayName == "" || len(x.Password) < 12 {
		httpx.Error(w, 422, "VALIDATION", "email, display name and password >= 12 are required")
		return
	}
	tx, err := a.social.DB.BeginTx(r.Context(), nil)
	if err != nil {
		httpx.Error(w, 500, "SETUP_FAILED", "unable to start setup")
		return
	}
	defer tx.Rollback()
	// Serialize bootstrap attempts so two concurrent requests cannot create two roots.
	if _, err = tx.ExecContext(r.Context(), `SELECT pg_advisory_xact_lock(821734991)`); err != nil {
		httpx.Error(w, 500, "SETUP_LOCK_FAILED", "unable to lock setup")
		return
	}
	var admins int
	if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM users WHERE role IN ('ADMIN_ROOT','ADMIN')`).Scan(&admins); err != nil {
		httpx.Error(w, 500, "SETUP_FAILED", "unable to inspect admin state")
		return
	}
	if admins > 0 {
		httpx.Error(w, 409, "SETUP_ALREADY_COMPLETE", "Admin Root already exists")
		return
	}
	// Register hashes the password with bcrypt. The insert is done through the same DB
	// transaction after the uniqueness check to keep the one-time bootstrap atomic.
	u, err := a.social.Register(r.Context(), "root", x.Email, x.Password, x.DisplayName, "PERSONAL")
	if err != nil {
		httpx.Error(w, 409, "BOOTSTRAP_REGISTER_FAILED", "unable to create root account")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE users SET role='ADMIN_ROOT',verified=true WHERE id=$1`, u.ID); err != nil {
		httpx.Error(w, 500, "BOOTSTRAP_ROLE_FAILED", "unable to promote root")
		return
	}
	if err = tx.Commit(); err != nil {
		httpx.Error(w, 500, "BOOTSTRAP_COMMIT_FAILED", "unable to complete bootstrap")
		return
	}
	httpx.JSON(w, 201, map[string]any{"ok": true, "user_id": u.ID, "role": "ADMIN_ROOT"})
}
