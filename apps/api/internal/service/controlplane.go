package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

type ControlPlane struct{ DB *sql.DB }

func (c ControlPlane) Roles(ctx context.Context) ([]map[string]any, error) {
	rows, err := c.DB.QueryContext(ctx, `SELECT name,permissions::text FROM admin_roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var name, perms string
		if err := rows.Scan(&name, &perms); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"name": name, "permissions": perms})
	}
	return out, rows.Err()
}

func (c ControlPlane) AdminUsers(ctx context.Context) ([]map[string]any, error) {
	rows, err := c.DB.QueryContext(ctx, `SELECT u.id::text,u.username,u.email,u.display_name,u.role,u.verified,u.created_at::text,COALESCE(d.id::text,''),COALESCE(d.code,''),COALESCE(d.name,''),COALESCE(uda.job_title,'') FROM users u LEFT JOIN user_department_assignments uda ON uda.user_id=u.id LEFT JOIN organization_departments d ON d.id=uda.department_id WHERE u.role <> 'USER' ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, user, email, name, role, created, deptID, deptCode, deptName, jobTitle string
		var verified bool
		if err := rows.Scan(&id, &user, &email, &name, &role, &verified, &created, &deptID, &deptCode, &deptName, &jobTitle); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "username": user, "email": email, "displayName": name, "role": role, "verified": verified, "createdAt": created, "departmentId": deptID, "departmentCode": deptCode, "department": deptName, "jobTitle": jobTitle})
	}
	return out, rows.Err()
}

func (c ControlPlane) GrantRole(ctx context.Context, actor, userID, role string, permissions []string) error {
	if actor == userID {
		return fmt.Errorf("root cannot alter its own control role")
	}
	role = strings.ToUpper(strings.TrimSpace(role))
	if role == "ADMIN_ROOT" {
		return fmt.Errorf("ADMIN_ROOT can only be established by bootstrap/recovery procedure")
	}
	var catalogPerms string
	if err := c.DB.QueryRowContext(ctx, `SELECT permissions::text FROM admin_roles WHERE name=$1`, role).Scan(&catalogPerms); err != nil {
		return fmt.Errorf("role is not in the approved role catalog")
	}
	if len(permissions) == 0 {
		if err := json.Unmarshal([]byte(catalogPerms), &permissions); err != nil {
			return fmt.Errorf("invalid role catalog permissions")
		}
	}
	var exists bool
	if err := c.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("user not found")
	}
	_, err := c.DB.ExecContext(ctx, `UPDATE users SET role=$1 WHERE id=$2`, role, userID)
	if err != nil {
		return err
	}
	_, err = c.DB.ExecContext(ctx, `INSERT INTO admin_role_assignments(user_id,role_name,permissions,granted_by) VALUES($1,$2,$3::jsonb,$4) ON CONFLICT(user_id) DO UPDATE SET role_name=EXCLUDED.role_name,permissions=EXCLUDED.permissions,granted_by=EXCLUDED.granted_by,updated_at=now()`, userID, role, JSON(permissions), actor)
	if err != nil {
		return err
	}
	_, err = c.DB.ExecContext(ctx, `INSERT INTO audit_logs(actor_id,action,target_type,target_id,metadata) VALUES($1,'ROLE_GRANTED','USER',$2,jsonb_build_object('role',$3))`, actor, userID, role)
	return err
}

func (c ControlPlane) RevokeRole(ctx context.Context, actor, userID string) error {
	if actor == userID {
		return fmt.Errorf("root cannot revoke its own role")
	}
	if _, err := c.DB.ExecContext(ctx, `UPDATE users SET role='USER' WHERE id=$1 AND role <> 'ADMIN_ROOT'`, userID); err != nil {
		return err
	}
	_, err := c.DB.ExecContext(ctx, `DELETE FROM admin_role_assignments WHERE user_id=$1`, userID)
	if err != nil {
		return err
	}
	_, err = c.DB.ExecContext(ctx, `INSERT INTO audit_logs(actor_id,action,target_type,target_id,metadata) VALUES($1,'ROLE_REVOKED','USER',$2,'{}'::jsonb)`, actor, userID)
	return err
}

func (c ControlPlane) Integrations(ctx context.Context) ([]map[string]any, error) {
	rows, err := c.DB.QueryContext(ctx, `SELECT id::text,name,provider,status,updated_at::text FROM platform_integrations ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, provider, status, updated string
		if err := rows.Scan(&id, &name, &provider, &status, &updated); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "name": name, "provider": provider, "status": status, "updatedAt": updated})
	}
	return out, rows.Err()
}
func (c ControlPlane) UpsertIntegration(ctx context.Context, id, name, provider, status string, config map[string]any) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(provider) == "" {
		return fmt.Errorf("name and provider are required")
	}
	if id == "" {
		id = uuid.New().String()
	}
	_, err := c.DB.ExecContext(ctx, `INSERT INTO platform_integrations(id,name,provider,status,config) VALUES($1,$2,$3,$4,$5::jsonb) ON CONFLICT(name) DO UPDATE SET provider=EXCLUDED.provider,status=EXCLUDED.status,config=EXCLUDED.config,updated_at=now()`, id, name, provider, status, JSON(config))
	return err
}

func (c ControlPlane) Deployments(ctx context.Context) ([]map[string]any, error) {
	rows, err := c.DB.QueryContext(ctx, `SELECT id::text,target,environment,version,provider,status,message,created_at::text FROM deployment_jobs ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, target, env, ver, provider, status, msg, created string
		if err := rows.Scan(&id, &target, &env, &ver, &provider, &status, &msg, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "target": target, "environment": env, "version": ver, "provider": provider, "status": status, "message": msg, "createdAt": created})
	}
	return out, rows.Err()
}
func (c ControlPlane) RequestDeployment(ctx context.Context, actor, target, env, version, provider string) (map[string]any, error) {
	if target == "" || env == "" || version == "" || provider == "" {
		return nil, fmt.Errorf("target, environment, version and provider are required")
	}
	id := uuid.New()
	msg := "Queued for provider orchestration"
	_, err := c.DB.ExecContext(ctx, `INSERT INTO deployment_jobs(id,target,environment,version,provider,status,message,requested_by) VALUES($1,$2,$3,$4,$5,'PENDING',$6,$7)`, id, target, env, version, provider, msg, actor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id.String(), "status": "PENDING", "message": msg}, nil
}

func (c ControlPlane) MobileBuilds(ctx context.Context) ([]map[string]any, error) {
	rows, err := c.DB.QueryContext(ctx, `SELECT id::text,platform,version,build_number,environment,provider,status,store_track,message,created_at::text FROM mobile_builds ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, platform, version, bn, env, provider, status, track, msg, created string
		if err := rows.Scan(&id, &platform, &version, &bn, &env, &provider, &status, &track, &msg, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "platform": platform, "version": version, "buildNumber": bn, "environment": env, "provider": provider, "status": status, "storeTrack": track, "message": msg, "createdAt": created})
	}
	return out, rows.Err()
}
func (c ControlPlane) RequestMobileBuild(ctx context.Context, actor, platform, version, bn, env, provider, track string) (map[string]any, error) {
	platform = strings.ToUpper(platform)
	if platform != "ANDROID" && platform != "IOS" {
		return nil, fmt.Errorf("platform must be ANDROID or IOS")
	}
	if version == "" || bn == "" || env == "" {
		return nil, fmt.Errorf("version, build number and environment are required")
	}
	if provider == "" {
		provider = "EAS"
	}
	if track == "" {
		track = "INTERNAL"
	}
	id := uuid.New()
	msg := "Queued for mobile build provider"
	_, err := c.DB.ExecContext(ctx, `INSERT INTO mobile_builds(id,platform,version,build_number,environment,provider,status,store_track,message,requested_by) VALUES($1,$2,$3,$4,$5,$6,'PENDING',$7,$8,$9)`, id, platform, version, bn, env, provider, track, msg, actor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id.String(), "status": "PENDING", "message": msg}, nil
}

func (c ControlPlane) AssignDepartment(ctx context.Context, actor, userID, departmentID, jobTitle string) error {
	if actor == userID {
		return fmt.Errorf("root cannot alter its own department assignment")
	}
	var exists bool
	if err := c.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, userID).Scan(&exists); err != nil || !exists {
		return fmt.Errorf("user not found")
	}
	if err := c.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organization_departments WHERE id=$1 AND status='ACTIVE')`, departmentID).Scan(&exists); err != nil || !exists {
		return fmt.Errorf("department not found")
	}
	_, err := c.DB.ExecContext(ctx, `INSERT INTO user_department_assignments(user_id,department_id,job_title,granted_by) VALUES($1,$2,$3,$4) ON CONFLICT(user_id) DO UPDATE SET department_id=EXCLUDED.department_id,job_title=EXCLUDED.job_title,granted_by=EXCLUDED.granted_by,updated_at=now()`, userID, departmentID, strings.TrimSpace(jobTitle), actor)
	if err != nil {
		return err
	}
	_, err = c.DB.ExecContext(ctx, `INSERT INTO audit_logs(actor_id,action,target_type,target_id,metadata) VALUES($1,'DEPARTMENT_ASSIGNED','USER',$2,jsonb_build_object('department_id',$3,'job_title',$4))`, actor, userID, departmentID, jobTitle)
	return err
}
