package httpapi

import (
	"github.com/nusamedia/platform/apps/api/internal/httpx"
	"github.com/nusamedia/platform/apps/api/internal/middleware"
	"net/http"
)

func (a *Router) controlRoles(w http.ResponseWriter, r *http.Request) {
	v, e := a.control.Roles(r.Context())
	if e != nil {
		httpx.Error(w, 500, "CONTROL_ROLES_FAILED", "unable to load roles")
		return
	}
	httpx.JSON(w, 200, map[string]any{"roles": v})
}
func (a *Router) controlUsers(w http.ResponseWriter, r *http.Request) {
	v, e := a.control.AdminUsers(r.Context())
	if e != nil {
		httpx.Error(w, 500, "CONTROL_USERS_FAILED", "unable to load admin users")
		return
	}
	httpx.JSON(w, 200, map[string]any{"users": v})
}
func (a *Router) controlGrantRole(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if e := a.control.GrantRole(r.Context(), middleware.Claims(r).UserID, r.PathValue("id"), x.Role, x.Permissions); e != nil {
		httpx.Error(w, 422, "ROLE_GRANT_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"ok": true})
}
func (a *Router) controlAssignDepartment(w http.ResponseWriter, r *http.Request) {
	var x struct {
		DepartmentID string `json:"departmentId"`
		JobTitle     string `json:"jobTitle"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if e := a.control.AssignDepartment(r.Context(), middleware.Claims(r).UserID, r.PathValue("id"), x.DepartmentID, x.JobTitle); e != nil {
		httpx.Error(w, 422, "DEPARTMENT_ASSIGN_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"ok": true})
}
func (a *Router) controlRevokeRole(w http.ResponseWriter, r *http.Request) {
	if e := a.control.RevokeRole(r.Context(), middleware.Claims(r).UserID, r.PathValue("id")); e != nil {
		httpx.Error(w, 422, "ROLE_REVOKE_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"ok": true})
}
func (a *Router) controlIntegrations(w http.ResponseWriter, r *http.Request) {
	v, e := a.control.Integrations(r.Context())
	if e != nil {
		httpx.Error(w, 500, "CONTROL_INTEGRATIONS_FAILED", "unable to load integrations")
		return
	}
	httpx.JSON(w, 200, map[string]any{"integrations": v})
}
func (a *Router) controlUpsertIntegration(w http.ResponseWriter, r *http.Request) {
	var x struct {
		ID, Name, Provider, Status string
		Config                     map[string]any `json:"config"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if e := a.control.UpsertIntegration(r.Context(), x.ID, x.Name, x.Provider, x.Status, x.Config); e != nil {
		httpx.Error(w, 422, "INTEGRATION_SAVE_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"ok": true})
}
func (a *Router) controlDeployments(w http.ResponseWriter, r *http.Request) {
	v, e := a.control.Deployments(r.Context())
	if e != nil {
		httpx.Error(w, 500, "CONTROL_DEPLOYMENTS_FAILED", "unable to load deployments")
		return
	}
	httpx.JSON(w, 200, map[string]any{"deployments": v})
}
func (a *Router) controlRequestDeployment(w http.ResponseWriter, r *http.Request) {
	var x struct{ Target, Environment, Version, Provider string }
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	v, e := a.control.RequestDeployment(r.Context(), middleware.Claims(r).UserID, x.Target, x.Environment, x.Version, x.Provider)
	if e != nil {
		httpx.Error(w, 422, "DEPLOYMENT_REQUEST_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 202, v)
}
func (a *Router) controlMobileBuilds(w http.ResponseWriter, r *http.Request) {
	v, e := a.control.MobileBuilds(r.Context())
	if e != nil {
		httpx.Error(w, 500, "CONTROL_MOBILE_FAILED", "unable to load mobile builds")
		return
	}
	httpx.JSON(w, 200, map[string]any{"builds": v})
}
func (a *Router) controlRequestMobileBuild(w http.ResponseWriter, r *http.Request) {
	var x struct{ Platform, Version, BuildNumber, Environment, Provider, StoreTrack string }
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	v, e := a.control.RequestMobileBuild(r.Context(), middleware.Claims(r).UserID, x.Platform, x.Version, x.BuildNumber, x.Environment, x.Provider, x.StoreTrack)
	if e != nil {
		httpx.Error(w, 422, "MOBILE_BUILD_REQUEST_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 202, v)
}

func (a *Router) workspaceDepartments(w http.ResponseWriter, r *http.Request) {
	v, e := a.workspace.Departments(r.Context())
	if e != nil {
		httpx.Error(w, 500, "WORKSPACE_DEPARTMENTS_FAILED", "unable to load departments")
		return
	}
	httpx.JSON(w, 200, map[string]any{"departments": v})
}
func (a *Router) workspaces(w http.ResponseWriter, r *http.Request) {
	v, e := a.workspace.Workspaces(r.Context())
	if e != nil {
		httpx.Error(w, 500, "WORKSPACES_FAILED", "unable to load workspaces")
		return
	}
	httpx.JSON(w, 200, map[string]any{"workspaces": v})
}
func (a *Router) workspaceChannels(w http.ResponseWriter, r *http.Request) {
	v, e := a.workspace.Channels(r.Context(), r.PathValue("id"))
	if e != nil {
		httpx.Error(w, 500, "WORKSPACE_CHANNELS_FAILED", "unable to load channels")
		return
	}
	httpx.JSON(w, 200, map[string]any{"channels": v})
}
func (a *Router) workspaceMembers(w http.ResponseWriter, r *http.Request) {
	v, e := a.workspace.Members(r.Context(), r.PathValue("id"))
	if e != nil {
		httpx.Error(w, 500, "WORKSPACE_MEMBERS_FAILED", "unable to load members")
		return
	}
	httpx.JSON(w, 200, map[string]any{"members": v})
}
func (a *Router) workspaceCreateChannel(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Visibility string `json:"visibility"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if x.Type == "" {
		x.Type = "GENERAL"
	}
	if x.Visibility == "" {
		x.Visibility = "PRIVATE"
	}
	v, e := a.workspace.CreateChannel(r.Context(), r.PathValue("id"), x.Name, x.Type, x.Visibility)
	if e != nil {
		httpx.Error(w, 422, "CHANNEL_CREATE_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 201, v)
}
func (a *Router) workspaceAddMember(w http.ResponseWriter, r *http.Request) {
	var x struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if x.Role == "" {
		x.Role = "MEMBER"
	}
	if e := a.workspace.AddMember(r.Context(), r.PathValue("id"), x.UserID, x.Role, middleware.Claims(r).UserID); e != nil {
		httpx.Error(w, 422, "MEMBER_ADD_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"ok": true})
}
func (a *Router) workspaceMessage(w http.ResponseWriter, r *http.Request) {
	var x struct {
		ChannelID string `json:"channelId"`
		Body      string `json:"body"`
		Priority  string `json:"priority"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	if x.Priority == "" {
		x.Priority = "NORMAL"
	}
	v, e := a.workspace.PostMessage(r.Context(), r.PathValue("id"), x.ChannelID, middleware.Claims(r).UserID, x.Body, x.Priority)
	if e != nil {
		httpx.Error(w, 422, "WORKSPACE_MESSAGE_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 201, v)
}
func (a *Router) workspaceNotifications(w http.ResponseWriter, r *http.Request) {
	v, e := a.workspace.Notifications(r.Context(), middleware.Claims(r).UserID)
	if e != nil {
		httpx.Error(w, 500, "WORKSPACE_NOTIFICATIONS_FAILED", "unable to load workspace notifications")
		return
	}
	httpx.JSON(w, 200, map[string]any{"notifications": v})
}
func (a *Router) workspaceNotificationRead(w http.ResponseWriter, r *http.Request) {
	if e := a.workspace.ReadNotification(r.Context(), middleware.Claims(r).UserID, r.PathValue("id")); e != nil {
		httpx.Error(w, 422, "WORKSPACE_NOTIFICATION_READ_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, map[string]any{"ok": true})
}

func (a *Router) workflowCases(w http.ResponseWriter, r *http.Request) {
	v, e := a.workflow.Cases(r.Context(), r.URL.Query().Get("status"))
	if e != nil {
		httpx.Error(w, 500, "WORKFLOW_CASES_FAILED", "unable to load workflow cases")
		return
	}
	httpx.JSON(w, 200, map[string]any{"cases": v})
}

func (a *Router) workflowTasks(w http.ResponseWriter, r *http.Request) {
	v, e := a.workflow.Tasks(r.Context(), r.PathValue("id"))
	if e != nil {
		httpx.Error(w, 500, "WORKFLOW_TASKS_FAILED", "unable to load workflow tasks")
		return
	}
	httpx.JSON(w, 200, map[string]any{"tasks": v})
}

func (a *Router) workflowDecision(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Decision string   `json:"decision"`
		Notes    string   `json:"notes"`
		Evidence []string `json:"evidence"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	v, e := a.workflow.DecideTask(r.Context(), middleware.Claims(r).UserID, r.PathValue("id"), r.PathValue("taskId"), x.Decision, x.Notes, x.Evidence)
	if e != nil {
		httpx.Error(w, 422, "WORKFLOW_DECISION_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, v)
}

func (a *Router) workflowApprove(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if httpx.Decode(r, &x) != nil {
		httpx.Error(w, 400, "INVALID_JSON", "invalid request")
		return
	}
	v, e := a.workflow.ApproveCase(r.Context(), middleware.Claims(r).UserID, r.PathValue("id"), x.Decision, x.Reason)
	if e != nil {
		httpx.Error(w, 422, "WORKFLOW_APPROVAL_FAILED", e.Error())
		return
	}
	httpx.JSON(w, 200, v)
}
