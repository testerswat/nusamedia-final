package httpapi

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/nusamedia/platform/apps/api/internal/auth"
	"github.com/nusamedia/platform/apps/api/internal/httpx"
	"github.com/nusamedia/platform/apps/api/internal/middleware"
	"github.com/nusamedia/platform/apps/api/internal/service"
	"golang.org/x/crypto/bcrypt"
)

type Router struct {
	social    service.Social
	platform  service.Platform
	control   service.ControlPlane
	workspace service.WorkspaceService
	workflow  service.WorkflowService
	secret    string
}

func New(_ any, social service.Social, secret string) *Router {
	return &Router{social: social, platform: service.Platform{DB: social.DB}, control: service.ControlPlane{DB: social.DB}, workspace: service.WorkspaceService{DB: social.DB}, workflow: service.WorkflowService{DB: social.DB}, secret: secret}
}
func (a *Router) Handler() *http.ServeMux {
	m := http.NewServeMux()
	health := func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, 200, map[string]any{"ok": true, "service": "nusamedia-api"})
	}
	m.HandleFunc("GET /health", health)
	m.HandleFunc("GET /api/health", health)
	m.HandleFunc("GET /api/v1/setup", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, 200, map[string]any{"status": "ready", "mode": "production-foundation", "message": "Use installer bootstrap before opening admin root."})
	})
	m.HandleFunc("POST /api/v1/setup/bootstrap", a.bootstrapRoot)
	m.HandleFunc("POST /api/v1/auth/register", a.register)
	m.HandleFunc("POST /api/v1/auth/login", a.login)
	m.HandleFunc("GET /api/v1/discover/nearby", a.discoverNearby)
	m.HandleFunc("GET /api/v1/public/settings", a.publicSettings)
	m.Handle("GET /api/v1/admin/settings", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.adminSettings))))
	m.Handle("PUT /api/v1/admin/settings", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.updateAdminSettings))))
	m.Handle("GET /api/v1/feed", middleware.Require(a.secret, http.HandlerFunc(a.feed)))
	m.Handle("POST /api/v1/posts", middleware.Require(a.secret, http.HandlerFunc(a.createPost)))
	m.Handle("POST /api/v1/posts/{id}/like", middleware.Require(a.secret, http.HandlerFunc(a.like)))
	m.Handle("POST /api/v1/posts/{id}/comments", middleware.Require(a.secret, http.HandlerFunc(a.comment)))
	m.Handle("POST /api/v1/users/{username}/follow", middleware.Require(a.secret, http.HandlerFunc(a.follow)))
	m.Handle("GET /api/v1/profile", middleware.Require(a.secret, http.HandlerFunc(a.profile)))
	m.Handle("PUT /api/v1/profile", middleware.Require(a.secret, http.HandlerFunc(a.updateProfile)))
	m.Handle("GET /api/v1/notifications", middleware.Require(a.secret, http.HandlerFunc(a.notifications)))
	m.Handle("GET /api/v1/posts/{id}/comments", middleware.Require(a.secret, http.HandlerFunc(a.comments)))
	m.Handle("POST /api/v1/posts/{id}/save", middleware.Require(a.secret, http.HandlerFunc(a.save)))
	m.Handle("POST /api/v1/posts/{id}/share", middleware.Require(a.secret, http.HandlerFunc(a.share)))
	m.Handle("GET /api/v1/search", middleware.Require(a.secret, http.HandlerFunc(a.search)))
	m.Handle("POST /api/v1/verification", middleware.Require(a.secret, http.HandlerFunc(a.verification)))
	m.Handle("GET /api/v1/admin/workflows", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workflowCases))))
	m.Handle("GET /api/v1/admin/workflows/{id}/tasks", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workflowTasks))))
	m.Handle("POST /api/v1/admin/workflows/{id}/tasks/{taskId}/decision", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workflowDecision))))
	m.Handle("POST /api/v1/admin/workflows/{id}/approve", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workflowApprove))))
	m.Handle("GET /api/v1/products", http.HandlerFunc(a.products))
	m.Handle("POST /api/v1/orders", middleware.Require(a.secret, http.HandlerFunc(a.order)))
	m.Handle("POST /api/v1/orders/{id}/pay", middleware.Require(a.secret, http.HandlerFunc(a.payOrder)))
	m.Handle("GET /api/v1/wallet", middleware.Require(a.secret, http.HandlerFunc(a.wallet)))
	m.Handle("POST /api/v1/wallet/topup", middleware.Require(a.secret, http.HandlerFunc(a.topup)))
	m.Handle("GET /api/v1/admin/dashboard", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.dashboard))))
	m.Handle("GET /api/v1/admin/control/roles", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.controlRoles))))
	m.Handle("GET /api/v1/admin/control/users", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.controlUsers))))
	m.Handle("POST /api/v1/admin/control/users/{id}/role", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.controlGrantRole))))
	m.Handle("POST /api/v1/admin/control/users/{id}/department", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.controlAssignDepartment))))
	m.Handle("DELETE /api/v1/admin/control/users/{id}/role", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.controlRevokeRole))))
	m.Handle("GET /api/v1/admin/control/integrations", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.controlIntegrations))))
	m.Handle("PUT /api/v1/admin/control/integrations", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.controlUpsertIntegration))))
	m.Handle("GET /api/v1/admin/control/deployments", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.controlDeployments))))
	m.Handle("POST /api/v1/admin/control/deployments", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.controlRequestDeployment))))
	m.Handle("GET /api/v1/admin/control/mobile/builds", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.controlMobileBuilds))))
	m.Handle("POST /api/v1/admin/control/mobile/builds", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.controlRequestMobileBuild))))

	// Organization and stakeholder workspace APIs. Membership and messaging are scoped to a workspace.
	m.Handle("GET /api/v1/admin/workspaces/departments", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workspaceDepartments))))
	m.Handle("GET /api/v1/admin/workspaces", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workspaces))))
	m.Handle("GET /api/v1/admin/workspaces/{id}/channels", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workspaceChannels))))
	m.Handle("GET /api/v1/admin/workspaces/{id}/members", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workspaceMembers))))
	m.Handle("POST /api/v1/admin/workspaces/{id}/channels", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.workspaceCreateChannel))))
	m.Handle("POST /api/v1/admin/workspaces/{id}/members", middleware.Require(a.secret, middleware.RequireRootDB(a.social.DB, http.HandlerFunc(a.workspaceAddMember))))
	m.Handle("POST /api/v1/admin/workspaces/{id}/messages", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workspaceMessage))))
	m.Handle("GET /api/v1/admin/workspace-notifications", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workspaceNotifications))))
	m.Handle("POST /api/v1/admin/workspace-notifications/{id}/read", middleware.Require(a.secret, middleware.RequireAdminDB(a.social.DB, http.HandlerFunc(a.workspaceNotificationRead))))

	return middleware.Security(withCORS(m))
}
func (a *Router) register(w http.ResponseWriter, r *http.Request) {
	var x struct{ Username, Email, Password, DisplayName string }
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if x.Username == "" || x.Email == "" || len(x.Password) < 8 || x.DisplayName == "" {
		httpx.Error(w, 422, "VALIDATION", "username,email,display_name and password >= 8 are required")
		return
	}
	u, e := a.social.Register(r.Context(), x.Username, x.Email, x.Password, x.DisplayName, "PERSONAL")
	if e != nil {
		httpx.Error(w, 409, "REGISTER_FAILED", e.Error())
		return
	}
	t, _ := auth.Sign(a.secret, u.ID, u.Role)
	httpx.JSON(w, 201, map[string]any{"user": u, "access_token": t})
}
func (a *Router) login(w http.ResponseWriter, r *http.Request) {
	var x struct{ Identity, Password string }
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	u, e := a.social.Login(r.Context(), x.Identity, x.Password)
	if e != nil {
		httpx.Error(w, 401, "INVALID_CREDENTIALS", "invalid credentials")
		return
	}
	t, _ := auth.Sign(a.secret, u.ID, u.Role)
	httpx.JSON(w, 200, map[string]any{"user": u, "access_token": t})
}
func (a *Router) feed(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 20
	}
	p, e := a.social.Feed(r.Context(), middleware.Claims(r).UserID, limit)
	if e != nil {
		httpx.Error(w, 500, "FEED_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"items": p, "next_cursor": nil})
}
func (a *Router) createPost(w http.ResponseWriter, r *http.Request) {
	var x struct{ Caption, Visibility string }
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if x.Visibility == "" {
		x.Visibility = "PUBLIC"
	}
	p, e := a.social.CreatePost(r.Context(), middleware.Claims(r).UserID, x.Caption, x.Visibility)
	if e != nil {
		httpx.Error(w, 500, "CREATE_POST_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 201, map[string]any{"post": p})
}
func (a *Router) like(w http.ResponseWriter, r *http.Request) {
	liked, count, e := a.social.ToggleLike(r.Context(), r.PathValue("id"), middleware.Claims(r).UserID)
	if e != nil {
		httpx.Error(w, 500, "LIKE_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"liked": liked, "likes_count": count})
}
func (a *Router) comment(w http.ResponseWriter, r *http.Request) {
	var x struct{ Body string }
	if httpx.Decode(r, &x) != nil || x.Body == "" {
		httpx.Error(w, 422, "VALIDATION", "body is required")
		return
	}
	c, e := a.social.Comment(r.Context(), r.PathValue("id"), middleware.Claims(r).UserID, x.Body)
	if e != nil {
		httpx.Error(w, 500, "COMMENT_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 201, map[string]any{"comment": c})
}
func (a *Router) follow(w http.ResponseWriter, r *http.Request) {
	var id string
	e := a.social.DB.QueryRowContext(r.Context(), `SELECT id FROM users WHERE username=$1`, r.PathValue("username")).Scan(&id)
	if e != nil {
		httpx.Error(w, 404, "USER_NOT_FOUND", "user not found")
		return
	}
	following, e := a.social.Follow(r.Context(), middleware.Claims(r).UserID, id)
	if e != nil {
		httpx.Error(w, 500, "FOLLOW_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"following": following, "username": r.PathValue("username")})
}

func (a *Router) discoverNearby(w http.ResponseWriter, r *http.Request) {
	lat, e1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, e2 := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if e1 != nil || e2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		httpx.Error(w, 422, "LOCATION_REQUIRED", "valid lat and lon are required")
		return
	}
	radius, _ := strconv.Atoi(r.URL.Query().Get("radius"))
	if radius == 0 {
		radius = 3000
	}
	intent := strings.TrimSpace(r.URL.Query().Get("intent"))
	if intent == "" {
		intent = "sekitar sini"
	}
	items, label, err := a.social.DiscoverNearby(r.Context(), lat, lon, radius, intent, 18)
	if err != nil {
		httpx.Error(w, 502, "DISCOVERY_PROVIDER_FAILED", "nearby discovery provider unavailable")
		return
	}
	httpx.JSON(w, 200, map[string]any{"intent": intent, "title": label, "radius_meters": radius, "source": "OpenStreetMap", "items": items})
}

func withCORS(next http.Handler) http.Handler {
	allowed := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, candidate := range allowed {
				if strings.TrimSpace(candidate) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					break
				}
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Router) publicSettings(w http.ResponseWriter, r *http.Request) {
	s, err := a.social.GetPublicSettings(r.Context())
	if err != nil {
		httpx.Error(w, 500, "SETTINGS_FAILED", "settings unavailable")
		return
	}
	httpx.JSON(w, 200, map[string]any{"settings": map[string]string{"hero": s.HeroImage, "footer": s.FooterImage}})
}

func (a *Router) adminSettings(w http.ResponseWriter, r *http.Request) {
	s, err := a.social.GetPublicSettings(r.Context())
	if err != nil {
		httpx.Error(w, 500, "SETTINGS_FAILED", "settings unavailable")
		return
	}
	httpx.JSON(w, 200, map[string]any{"settings": map[string]string{"hero": s.HeroImage, "footer": s.FooterImage}})
}

func (a *Router) updateAdminSettings(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Hero   string `json:"hero"`
		Footer string `json:"footer"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if x.Hero != "" {
		if err := a.social.UpdateMediaSetting(r.Context(), "hero_image", x.Hero); err != nil {
			httpx.Error(w, 422, "INVALID_HERO", "invalid hero image")
			return
		}
	}
	if x.Footer != "" {
		if err := a.social.UpdateMediaSetting(r.Context(), "footer_image", x.Footer); err != nil {
			httpx.Error(w, 422, "INVALID_FOOTER", "invalid footer image")
			return
		}
	}
	a.publicSettings(w, r)
}

func (a *Router) profile(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.Profile(r.Context(), middleware.Claims(r).UserID)
	if e != nil {
		httpx.Error(w, 404, "PROFILE_NOT_FOUND", "profile not found")
		return
	}
	httpx.JSON(w, 200, map[string]any{"profile": v})
}
func (a *Router) updateProfile(w http.ResponseWriter, r *http.Request) {
	var x map[string]string
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if e := a.platform.UpdateProfile(r.Context(), middleware.Claims(r).UserID, x); e != nil {
		httpx.Error(w, 422, "PROFILE_UPDATE_FAILED", e.Error())
		return
	}
	a.profile(w, r)
}
func (a *Router) notifications(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.Notifications(r.Context(), middleware.Claims(r).UserID)
	if e != nil {
		httpx.Error(w, 500, "NOTIFICATIONS_FAILED", "notifications unavailable")
		return
	}
	httpx.JSON(w, 200, map[string]any{"items": v})
}
func (a *Router) comments(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.Comments(r.Context(), r.PathValue("id"))
	if e != nil {
		httpx.Error(w, 500, "COMMENTS_FAILED", "comments unavailable")
		return
	}
	httpx.JSON(w, 200, map[string]any{"items": v})
}
func (a *Router) save(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.Save(r.Context(), r.PathValue("id"), middleware.Claims(r).UserID)
	if e != nil {
		httpx.Error(w, 500, "SAVE_FAILED", "save failed")
		return
	}
	httpx.JSON(w, 200, map[string]any{"saved": v})
}
func (a *Router) share(w http.ResponseWriter, r *http.Request) {
	e := a.platform.Share(r.Context(), r.PathValue("id"), middleware.Claims(r).UserID)
	if e != nil {
		httpx.Error(w, 500, "SHARE_FAILED", "share failed")
		return
	}
	httpx.JSON(w, 200, map[string]any{"shared": true})
}
func (a *Router) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		httpx.Error(w, 422, "QUERY_TOO_SHORT", "minimum 2 characters")
		return
	}
	v, e := a.platform.Search(r.Context(), q)
	if e != nil {
		httpx.Error(w, 500, "SEARCH_FAILED", "search unavailable")
		return
	}
	httpx.JSON(w, 200, map[string]any{"items": v})
}
func (a *Router) verification(w http.ResponseWriter, r *http.Request) {
	var x struct{ RequestedType, DocumentURL string }
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	v, e := a.workflow.CreateVerificationCase(r.Context(), middleware.Claims(r).UserID, x.RequestedType, x.DocumentURL)
	if e != nil {
		httpx.Error(w, 422, "VERIFICATION_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 201, map[string]any{"verification": v})
}
func (a *Router) products(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.Products(r.Context())
	if e != nil {
		httpx.Error(w, 500, "PRODUCTS_FAILED", "products unavailable")
		return
	}
	httpx.JSON(w, 200, map[string]any{"items": v})
}
func (a *Router) order(w http.ResponseWriter, r *http.Request) {
	var x struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
	}
	if httpx.Decode(r, &x) != nil || x.ProductID == "" || x.Quantity < 1 {
		httpx.Error(w, 422, "VALIDATION", "productId and positive quantity required")
		return
	}
	v, e := a.platform.CreateOrder(r.Context(), middleware.Claims(r).UserID, x.ProductID, x.Quantity)
	if e != nil {
		httpx.Error(w, 422, "ORDER_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 201, map[string]any{"order": v})
}
func (a *Router) payOrder(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.PayOrder(r.Context(), middleware.Claims(r).UserID, r.PathValue("id"))
	if e != nil {
		httpx.Error(w, 422, "PAYMENT_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"payment": v})
}
func (a *Router) wallet(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.Wallet(r.Context(), middleware.Claims(r).UserID)
	if e != nil {
		httpx.Error(w, 500, "WALLET_FAILED", "wallet unavailable")
		return
	}
	httpx.JSON(w, 200, v)
}
func (a *Router) topup(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Amount        int64  `json:"amount"`
		PaymentMethod string `json:"paymentMethod"`
		EvidenceURL   string `json:"evidenceUrl"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	v, e := a.workflow.CreateTopupRequest(r.Context(), middleware.Claims(r).UserID, x.Amount, x.PaymentMethod, x.EvidenceURL)
	if e != nil {
		httpx.Error(w, 422, "TOPUP_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 201, v)
}
func (a *Router) dashboard(w http.ResponseWriter, r *http.Request) {
	v, e := a.platform.AdminDashboard(r.Context())
	if e != nil {
		httpx.Error(w, 500, "DASHBOARD_FAILED", "dashboard unavailable")
		return
	}
	httpx.JSON(w, 200, v)
}

func (a *Router) bootstrapRoot(w http.ResponseWriter, r *http.Request) {
	configured := strings.TrimSpace(os.Getenv("NUSAMEDIA_SETUP_TOKEN"))
	if configured == "" || r.Header.Get("X-NusaMedia-Setup-Token") != configured {
		httpx.Error(w, http.StatusForbidden, "SETUP_FORBIDDEN", "invalid or missing setup token")
		return
	}
	var x struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if strings.TrimSpace(x.Email) == "" || strings.TrimSpace(x.DisplayName) == "" || len(x.Password) < 12 {
		httpx.Error(w, 422, "VALIDATION", "email, display_name and password >= 12 are required")
		return
	}
	tx, err := a.social.DB.BeginTx(r.Context(), nil)
	if err != nil {
		httpx.Error(w, 500, "BOOTSTRAP_FAILED", "database unavailable")
		return
	}
	defer tx.Rollback()
	var locked bool
	if err = tx.QueryRowContext(r.Context(), `SELECT pg_try_advisory_xact_lock(hashtextextended('nusamedia.admin_root_bootstrap',0))`).Scan(&locked); err != nil || !locked {
		httpx.Error(w, 409, "BOOTSTRAP_BUSY", "bootstrap is busy")
		return
	}
	var count int
	if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM users WHERE role='ADMIN_ROOT'`).Scan(&count); err != nil {
		httpx.Error(w, 500, "BOOTSTRAP_FAILED", "unable to inspect admin root")
		return
	}
	if count > 0 {
		httpx.Error(w, 409, "ROOT_EXISTS", "Admin Root already exists")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(x.Password), bcrypt.DefaultCost)
	id := uuid.New()
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO users(id,username,email,password_hash,display_name,account_type,role,verified) VALUES($1,$2,$3,$4,$5,'PERSONAL','ADMIN_ROOT',true)`, id, "root_"+id.String()[:8], strings.ToLower(x.Email), string(h), x.DisplayName); err != nil {
		httpx.Error(w, 409, "BOOTSTRAP_FAILED", "unable to create Admin Root")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO admin_role_assignments(user_id,role_name,permissions,granted_by) VALUES($1,'ADMIN_ROOT','["*"]'::jsonb,$1)`, id); err != nil {
		httpx.Error(w, 500, "BOOTSTRAP_FAILED", "unable to initialize root authority")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO audit_logs(actor_id,action,target_type,target_id,metadata) VALUES($1,'ROOT_BOOTSTRAP','ADMIN_ROOT',$2,'{"source":"installer"}'::jsonb)`, id, id.String()); err != nil {
		httpx.Error(w, 500, "BOOTSTRAP_FAILED", "unable to write audit record")
		return
	}
	if err = tx.Commit(); err != nil {
		httpx.Error(w, 500, "BOOTSTRAP_FAILED", "unable to commit root")
		return
	}
	httpx.JSON(w, 201, map[string]any{"ok": true, "user_id": id.String(), "role": "ADMIN_ROOT"})
}
