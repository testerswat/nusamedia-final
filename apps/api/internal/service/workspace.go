package service

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

type WorkspaceService struct{ DB *sql.DB }

func (s WorkspaceService) Departments(ctx context.Context) ([]map[string]any, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id::text,code,name,description,status FROM organization_departments ORDER BY name`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, code, name, desc, status string
		if e := rows.Scan(&id, &code, &name, &desc, &status); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "code": code, "name": name, "description": desc, "status": status})
	}
	return out, rows.Err()
}
func (s WorkspaceService) Workspaces(ctx context.Context) ([]map[string]any, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT w.id::text,w.name,w.slug,w.description,w.visibility,w.status,COALESCE(d.name,'') FROM workspaces w LEFT JOIN organization_departments d ON d.id=w.department_id ORDER BY w.name`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, slug, desc, vis, status, dept string
		if e := rows.Scan(&id, &name, &slug, &desc, &vis, &status, &dept); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "name": name, "slug": slug, "description": desc, "visibility": vis, "status": status, "department": dept})
	}
	return out, rows.Err()
}
func (s WorkspaceService) Channels(ctx context.Context, workspaceID string) ([]map[string]any, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id::text,name,channel_type,visibility FROM workspace_channels WHERE workspace_id=$1 ORDER BY name`, workspaceID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, typ, vis string
		if e := rows.Scan(&id, &name, &typ, &vis); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "name": name, "type": typ, "visibility": vis})
	}
	return out, rows.Err()
}
func (s WorkspaceService) Members(ctx context.Context, workspaceID string) ([]map[string]any, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT u.id::text,u.username,u.display_name,u.email,wm.membership_role,wm.status FROM workspace_members wm JOIN users u ON u.id=wm.user_id WHERE wm.workspace_id=$1 ORDER BY u.display_name`, workspaceID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, user, name, email, role, status string
		if e := rows.Scan(&id, &user, &name, &email, &role, &status); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "username": user, "displayName": name, "email": email, "role": role, "status": status})
	}
	return out, rows.Err()
}
func (s WorkspaceService) CreateChannel(ctx context.Context, workspaceID, name, typ, visibility string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("channel name is required")
	}
	id := uuid.New()
	_, e := s.DB.ExecContext(ctx, `INSERT INTO workspace_channels(id,workspace_id,name,channel_type,visibility) VALUES($1,$2,$3,$4,$5)`, id, workspaceID, name, typ, visibility)
	if e != nil {
		return nil, e
	}
	return map[string]any{"id": id.String(), "name": name, "type": typ, "visibility": visibility}, nil
}
func (s WorkspaceService) AddMember(ctx context.Context, workspaceID, userID, role, actor string) error {
	if workspaceID == "" || userID == "" {
		return fmt.Errorf("workspace and user are required")
	}
	if role == "" {
		role = "MEMBER"
	}
	_, e := s.DB.ExecContext(ctx, `INSERT INTO workspace_members(workspace_id,user_id,membership_role,granted_by) VALUES($1,$2,$3,$4) ON CONFLICT(workspace_id,user_id) DO UPDATE SET membership_role=EXCLUDED.membership_role,status='ACTIVE',granted_by=EXCLUDED.granted_by,updated_at=now()`, workspaceID, userID, role, actor)
	return e
}
func (s WorkspaceService) PostMessage(ctx context.Context, workspaceID, channelID, sender, body, priority string) (map[string]any, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("message body is required")
	}
	if priority == "" {
		priority = "NORMAL"
	}
	id := uuid.New()
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var ok bool
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='ACTIVE') OR EXISTS(SELECT 1 FROM users WHERE id=$2 AND role='ADMIN_ROOT')`, workspaceID, sender).Scan(&ok); e != nil || !ok {
		if e != nil {
			return nil, e
		}
		return nil, fmt.Errorf("sender is not a workspace member")
	}
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_channels WHERE id=$1 AND workspace_id=$2)`, channelID, workspaceID).Scan(&ok); e != nil || !ok {
		if e != nil {
			return nil, e
		}
		return nil, fmt.Errorf("channel does not belong to workspace")
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO workspace_messages(id,channel_id,sender_id,body,priority) VALUES($1,$2,$3,$4,$5)`, id, channelID, sender, body, priority); e != nil {
		return nil, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO workspace_notifications(workspace_id,channel_id,sender_id,recipient_user_id,title,body,event_type,priority) SELECT $1,$2,$3,user_id,'Pesan workspace', $4,'WORKSPACE_MESSAGE',$5 FROM workspace_members WHERE workspace_id=$1 AND status='ACTIVE' AND user_id<>$3`, workspaceID, channelID, sender, body, priority)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return map[string]any{"id": id.String(), "status": "SENT", "notified": "WORKSPACE_MEMBERS"}, nil
}
func (s WorkspaceService) Notifications(ctx context.Context, userID string) ([]map[string]any, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id::text,workspace_id::text,title,body,event_type,priority,read_at,created_at::text FROM workspace_notifications WHERE recipient_user_id=$1 ORDER BY created_at DESC LIMIT 100`, userID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, wid, title, body, typ, priority, created string
		var read sql.NullString
		if e := rows.Scan(&id, &wid, &title, &body, &typ, &priority, &read, &created); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": id, "workspaceId": wid, "title": title, "body": body, "eventType": typ, "priority": priority, "readAt": read.String, "createdAt": created})
	}
	return out, rows.Err()
}
func (s WorkspaceService) ReadNotification(ctx context.Context, userID, notificationID string) error {
	_, e := s.DB.ExecContext(ctx, `UPDATE workspace_notifications SET read_at=COALESCE(read_at,now()) WHERE id=$1 AND recipient_user_id=$2`, notificationID, userID)
	return e
}
