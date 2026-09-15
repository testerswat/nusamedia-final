package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type WorkflowService struct{ DB *sql.DB }

func (s WorkflowService) notifyDepartment(ctx context.Context, tx *sql.Tx, deptCode, title, body, eventType, priority string, sender any) error {
	_, err := tx.ExecContext(ctx, `
      INSERT INTO workspace_notifications(workspace_id,channel_id,sender_id,recipient_user_id,title,body,event_type,priority)
      SELECT w.id,c.id,$2,wm.user_id,$3,$4,$5,$6
      FROM organization_departments d
      JOIN workspaces w ON w.department_id=d.id AND w.status='ACTIVE'
      JOIN workspace_channels c ON c.workspace_id=w.id AND c.name='General'
      JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.status='ACTIVE'
      WHERE d.code=$1`, deptCode, sender, title, body, eventType, priority)
	return err
}

func (s WorkflowService) notifyManagers(ctx context.Context, tx *sql.Tx, deptCode, title, body, eventType, priority string, sender any) error {
	_, err := tx.ExecContext(ctx, `
      INSERT INTO workspace_notifications(workspace_id,channel_id,sender_id,recipient_user_id,title,body,event_type,priority)
      SELECT w.id,c.id,$2,wm.user_id,$3,$4,$5,$6
      FROM organization_departments d
      JOIN workspaces w ON w.department_id=d.id AND w.status='ACTIVE'
      JOIN workspace_channels c ON c.workspace_id=w.id AND c.name='General'
      JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.status='ACTIVE'
      LEFT JOIN admin_role_assignments ara ON ara.user_id=wm.user_id
      WHERE d.code=$1 AND (wm.membership_role IN ('MANAGER','LEAD','OWNER') OR ara.role_name IN ('FINANCE_MANAGER','VERIFICATION_MANAGER','OPERATIONS_MANAGER','MANAGER','ADMIN_ROOT'))`, deptCode, sender, title, body, eventType, priority)
	return err
}

func (s WorkflowService) CreateTopupRequest(ctx context.Context, user string, amount int64, method, evidence string) (map[string]any, error) {
	if amount <= 0 || amount > 1000000000 {
		return nil, fmt.Errorf("invalid amount")
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case "BANK_TRANSFER":
	default:
		return nil, fmt.Errorf("unsupported payment method")
	}
	evidence = strings.TrimSpace(evidence)
	if len(evidence) > 2048 {
		return nil, fmt.Errorf("evidence url too long")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	caseID := uuid.New()
	topupID := uuid.New()
	var financeDept string
	if err = tx.QueryRowContext(ctx, `SELECT id::text FROM organization_departments WHERE code='FIN'`).Scan(&financeDept); err != nil {
		return nil, err
	}
	meta, _ := json.Marshal(map[string]any{"paymentMethod": method, "evidenceUrl": evidence})
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_cases(id,case_type,subject_user_id,source_ref,status,priority,amount,currency,metadata,assigned_department_id,created_by) VALUES($1,'WALLET_TOPUP',$2,$1,'OPEN','NORMAL',$3,'IDR',$4::jsonb,$5,$2)`, caseID, user, amount, string(meta), financeDept); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO topup_requests(id,case_id,user_id,amount,payment_method,evidence_url) VALUES($1,$2,$3,$4,$5,$6)`, topupID, caseID, user, amount, method, evidence); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_tasks(case_id,task_type,department_id,status) VALUES($1,'FINANCE_REVIEW',$2,'PENDING')`, caseID, financeDept); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_events(case_id,actor_id,event_type,to_status,payload) VALUES($1,$2,'TOPUP_SUBMITTED','OPEN',$3::jsonb)`, caseID, user, string(meta)); err != nil {
		return nil, err
	}
	if err = s.notifyDepartment(ctx, tx, "FIN", "Top up menunggu pemeriksaan", fmt.Sprintf("Pengguna mengajukan top up sebesar IDR %d melalui %s. Dana belum ditambahkan.", amount, method), "TOPUP_REVIEW_REQUIRED", nil); err != nil {
		return nil, err
	}
	if err = s.notifyManagers(ctx, tx, "FIN", "Top up baru — review tim", fmt.Sprintf("Ada pengajuan top up IDR %d yang masuk ke Finance dan memerlukan pemeriksaan.", amount), "TOPUP_MANAGER_NOTICE", nil); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"caseId": caseID.String(), "topupId": topupID.String(), "status": "PENDING_REVIEW", "fundsCredited": false}, nil
}

func (s WorkflowService) CreateVerificationCase(ctx context.Context, user, typ, doc string) (map[string]any, error) {
	typ = strings.ToUpper(strings.TrimSpace(typ))
	if typ == "" || typ == "PERSONAL" {
		return nil, fmt.Errorf("verification type required")
	}
	doc = strings.TrimSpace(doc)
	if doc == "" || len(doc) > 2048 {
		return nil, fmt.Errorf("document url required")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	caseID := uuid.New()
	var deptID string
	if err = tx.QueryRowContext(ctx, `SELECT id::text FROM organization_departments WHERE code='TRUST'`).Scan(&deptID); err != nil {
		return nil, err
	}
	var verificationID string
	err = tx.QueryRowContext(ctx, `INSERT INTO verification_requests(user_id,requested_type,document_url,status,workflow_case_id) VALUES($1,$2,$3,'PENDING',$4) ON CONFLICT(user_id) DO UPDATE SET requested_type=EXCLUDED.requested_type,document_url=EXCLUDED.document_url,status='PENDING',review_note='',workflow_case_id=EXCLUDED.workflow_case_id,updated_at=now() RETURNING id::text`, user, typ, doc, caseID).Scan(&verificationID)
	if err != nil {
		return nil, err
	}
	meta, _ := json.Marshal(map[string]any{"verificationId": verificationID, "requestedType": typ})
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_cases(id,case_type,subject_user_id,source_ref,status,priority,metadata,assigned_department_id,created_by) VALUES($1,'IDENTITY_VERIFICATION',$2,$3,'OPEN','HIGH',$4::jsonb,$5,$2)`, caseID, user, verificationID, string(meta), deptID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_tasks(case_id,task_type,department_id,status) VALUES($1,'DOCUMENT_REVIEW',$2,'PENDING')`, caseID, deptID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_events(case_id,actor_id,event_type,to_status,payload) VALUES($1,$2,'VERIFICATION_SUBMITTED','OPEN',$3::jsonb)`, caseID, user, string(meta)); err != nil {
		return nil, err
	}
	if err = s.notifyDepartment(ctx, tx, "TRUST", "Verifikasi identitas membutuhkan pemeriksaan", "Dokumen verifikasi baru masuk. Periksa keabsahan dan kesesuaian data sebelum keputusan.", "VERIFICATION_REVIEW_REQUIRED", "HIGH", nil); err != nil {
		return nil, err
	}
	if err = s.notifyManagers(ctx, tx, "TRUST", "Verifikasi baru — manajer mendapat notifikasi", "Ada pengajuan verifikasi yang masuk. Keputusan reviewer belum berarti persetujuan final.", "VERIFICATION_MANAGER_NOTICE", "HIGH", nil); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"caseId": caseID.String(), "verificationId": verificationID, "status": "PENDING", "verified": false}, nil
}

func (s WorkflowService) Cases(ctx context.Context, status string) ([]map[string]any, error) {
	q := `SELECT c.id::text,c.case_type,c.status,c.priority,COALESCE(c.amount,0),c.currency,c.source_ref,c.created_at::text,COALESCE(u.email,''),COALESCE(d.name,'') FROM workflow_cases c LEFT JOIN users u ON u.id=c.subject_user_id LEFT JOIN organization_departments d ON d.id=c.assigned_department_id`
	args := []any{}
	if status != "" {
		q += ` WHERE c.status=$1`
		args = append(args, strings.ToUpper(status))
	}
	q += ` ORDER BY c.created_at DESC LIMIT 200`
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, typ, st, pr, currency, src, created, email, dept string
		var amount int64
		if err = rows.Scan(&id, &typ, &st, &pr, &amount, &currency, &src, &created, &email, &dept); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "caseType": typ, "status": st, "priority": pr, "amount": amount, "currency": currency, "sourceRef": src, "createdAt": created, "subject": email, "department": dept})
	}
	return out, rows.Err()
}

func (s WorkflowService) DecideTask(ctx context.Context, actor, caseID, taskID, decision, notes string, evidence []string) (map[string]any, error) {
	decision = strings.ToUpper(strings.TrimSpace(decision))
	if decision != "PASS" && decision != "FAIL" && decision != "ESCALATE" {
		return nil, fmt.Errorf("unsupported decision")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var typ, caseStatus, taskType, taskStatus, deptCode, subjectUser string
	if err = tx.QueryRowContext(ctx, `SELECT c.case_type,c.status,t.task_type,t.status,d.code,c.subject_user_id::text FROM workflow_cases c JOIN workflow_tasks t ON t.case_id=c.id LEFT JOIN organization_departments d ON d.id=t.department_id WHERE c.id=$1 AND t.id=$2 FOR UPDATE`, caseID, taskID).Scan(&typ, &caseStatus, &taskType, &taskStatus, &deptCode, &subjectUser); err != nil {
		return nil, fmt.Errorf("case/task not found")
	}
	if caseStatus != "OPEN" {
		return nil, fmt.Errorf("case is not open")
	}
	if taskStatus != "PENDING" {
		return nil, fmt.Errorf("task is no longer pending")
	}
	var actorRole string
	if err = tx.QueryRowContext(ctx, `SELECT role FROM users WHERE id=$1`, actor).Scan(&actorRole); err != nil {
		return nil, fmt.Errorf("actor not found")
	}
	actorRole = strings.ToUpper(actorRole)
	if actorRole == "ADMIN_ROOT" {
		// Root may intervene under the same workflow rules, with audit evidence.
	} else if typ == "WALLET_TOPUP" && actorRole != "FINANCE_REVIEWER" && actorRole != "FINANCE_MANAGER" {
		return nil, fmt.Errorf("finance reviewer or manager role required")
	} else if typ == "IDENTITY_VERIFICATION" && actorRole != "VERIFICATION_REVIEWER" && actorRole != "VERIFICATION_MANAGER" {
		return nil, fmt.Errorf("verification reviewer or manager role required")
	}
	ev, _ := json.Marshal(evidence)
	if _, err = tx.ExecContext(ctx, `UPDATE workflow_tasks SET status='COMPLETED',decision=$1,notes=$2,evidence=$3::jsonb,assignee_user_id=$4,completed_at=now(),updated_at=now() WHERE id=$5`, decision, notes, string(ev), actor, taskID); err != nil {
		return nil, err
	}
	next := "OPEN"
	switch typ {
	case "WALLET_TOPUP":
		if taskType == "FINANCE_REVIEW" {
			if decision == "PASS" {
				next = "AWAITING_APPROVAL"
			} else if decision == "FAIL" {
				next = "REJECTED"
			} else {
				next = "ESCALATED"
			}
		}
	case "IDENTITY_VERIFICATION":
		if taskType == "DOCUMENT_REVIEW" {
			if decision == "PASS" {
				next = "AWAITING_MANAGER_APPROVAL"
			} else if decision == "FAIL" {
				next = "REJECTED"
			} else {
				next = "ESCALATED"
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE workflow_cases SET status=$1,updated_at=now(),resolved_at=CASE WHEN $1 IN ('REJECTED','COMPLETED') THEN now() ELSE resolved_at END WHERE id=$2`, next, caseID); err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"taskType": taskType, "decision": decision, "notes": notes})
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_events(case_id,actor_id,event_type,from_status,to_status,payload) VALUES($1,$2,'TASK_DECISION',$3,$4,$5::jsonb)`, caseID, actor, caseStatus, next, string(payload)); err != nil {
		return nil, err
	}
	if decision == "PASS" && typ == "WALLET_TOPUP" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_approvals(case_id,task_id,requested_by,status,reason) VALUES($1,$2,$3,'PENDING','Finance review passed; separate approval required before wallet credit')`, caseID, taskID, actor); err != nil {
			return nil, err
		}
		if err = s.notifyManagers(ctx, tx, "FIN", "Persetujuan Finance diperlukan", "Reviewer menyelesaikan pemeriksaan top up. Dana tetap belum dikreditkan sampai approval terpisah.", "TOPUP_APPROVAL_REQUIRED", "HIGH", actor); err != nil {
			return nil, err
		}
	}
	if decision == "PASS" && typ == "IDENTITY_VERIFICATION" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_approvals(case_id,task_id,requested_by,status,reason) VALUES($1,$2,$3,'PENDING','Document review passed; manager approval required')`, caseID, taskID, actor); err != nil {
			return nil, err
		}
		if err = s.notifyManagers(ctx, tx, "TRUST", "Persetujuan manajer verifikasi diperlukan", "Reviewer menyatakan dokumen lolos pemeriksaan awal. Persetujuan akhir memerlukan manager review dan policy checks.", "VERIFICATION_APPROVAL_REQUIRED", "HIGH", actor); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"caseId": caseID, "status": next, "decision": decision, "fundsCredited": false, "verified": false}, nil
}

func (s WorkflowService) ApproveCase(ctx context.Context, actor, caseID, decision, reason string) (map[string]any, error) {
	decision = strings.ToUpper(strings.TrimSpace(decision))
	if decision != "APPROVE" && decision != "REJECT" {
		return nil, fmt.Errorf("unsupported approval decision")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var typ, status, subject, dept string
	var amount int64
	if err = tx.QueryRowContext(ctx, `SELECT c.case_type,c.status,c.subject_user_id::text,COALESCE(d.code,''),COALESCE(c.amount,0) FROM workflow_cases c LEFT JOIN organization_departments d ON d.id=c.assigned_department_id WHERE c.id=$1 FOR UPDATE`, caseID).Scan(&typ, &status, &subject, &dept, &amount); err != nil {
		return nil, fmt.Errorf("case not found")
	}
	if status != "AWAITING_APPROVAL" && status != "AWAITING_MANAGER_APPROVAL" {
		return nil, fmt.Errorf("case is not awaiting approval")
	}
	var approvalID string
	if err = tx.QueryRowContext(ctx, `SELECT id::text FROM workflow_approvals WHERE case_id=$1 AND status='PENDING' ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, caseID).Scan(&approvalID); err != nil {
		return nil, fmt.Errorf("approval not found")
	}
	var role string
	if err = tx.QueryRowContext(ctx, `SELECT role FROM users WHERE id=$1`, actor).Scan(&role); err != nil {
		return nil, fmt.Errorf("actor not found")
	}
	role = strings.ToUpper(role)
	var previousReviewer string
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(assignee_user_id::text,'') FROM workflow_tasks WHERE case_id=$1 AND status='COMPLETED' ORDER BY completed_at DESC LIMIT 1`, caseID).Scan(&previousReviewer); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if role != "ADMIN_ROOT" {
		if previousReviewer == actor {
			return nil, fmt.Errorf("separation of duties: reviewer cannot approve the same case")
		}
		if typ == "WALLET_TOPUP" && role != "FINANCE_MANAGER" {
			return nil, fmt.Errorf("finance manager or admin root approval required")
		}
		if typ == "IDENTITY_VERIFICATION" && role != "VERIFICATION_MANAGER" {
			return nil, fmt.Errorf("verification manager or admin root approval required")
		}
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE workflow_approvals SET status=$1,approved_by=$2,reason=$3,decided_at=now() WHERE id=$4`, map[bool]string{true: "APPROVED", false: "REJECTED"}[decision == "APPROVE"], actor, reason, approvalID)
	}
	if err != nil {
		return nil, err
	}
	next := "REJECTED"
	if decision == "APPROVE" {
		next = "COMPLETED"
	}
	if _, err = tx.ExecContext(ctx, `UPDATE workflow_cases SET status=$1,updated_at=now(),resolved_at=now() WHERE id=$2`, next, caseID); err != nil {
		return nil, err
	}
	if typ == "WALLET_TOPUP" && decision == "APPROVE" {
		var userID string
		if err = tx.QueryRowContext(ctx, `SELECT user_id::text FROM topup_requests WHERE case_id=$1 AND status='PENDING_REVIEW' FOR UPDATE`, caseID).Scan(&userID); err != nil {
			return nil, fmt.Errorf("top up request is no longer pending")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_accounts(user_id,balance) VALUES($1,0) ON CONFLICT(user_id) DO NOTHING`, userID); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE wallet_accounts SET balance=balance+$1,updated_at=now() WHERE user_id=$2`, amount, userID); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(user_id,type,amount,description) VALUES($1,'TOPUP',$2,'Top up disetujui melalui workflow Finance')`, userID, amount); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE topup_requests SET status='APPROVED',review_note=$1,updated_at=now() WHERE case_id=$2`, reason, caseID); err != nil {
			return nil, err
		}
	} else if typ == "WALLET_TOPUP" {
		if _, err = tx.ExecContext(ctx, `UPDATE topup_requests SET status='REJECTED',review_note=$1,updated_at=now() WHERE case_id=$2`, reason, caseID); err != nil {
			return nil, err
		}
	} else if typ == "IDENTITY_VERIFICATION" {
		if decision == "APPROVE" {
			if _, err = tx.ExecContext(ctx, `UPDATE verification_requests SET status='APPROVED',review_note=$1,updated_at=now() WHERE workflow_case_id=$2`, reason, caseID); err != nil {
				return nil, err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE users SET verified=true WHERE id=$1`, subject); err != nil {
				return nil, err
			}
		} else {
			if _, err = tx.ExecContext(ctx, `UPDATE verification_requests SET status='REJECTED',review_note=$1,updated_at=now() WHERE workflow_case_id=$2`, reason, caseID); err != nil {
				return nil, err
			}
		}
	}
	payload, _ := json.Marshal(map[string]any{"decision": decision, "reason": reason, "role": role})
	if _, err = tx.ExecContext(ctx, `INSERT INTO workflow_events(case_id,actor_id,event_type,from_status,to_status,payload) VALUES($1,$2,'MANAGER_APPROVAL',$3,$4,$5::jsonb)`, caseID, actor, status, next, string(payload)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"caseId": caseID, "status": next, "decision": decision, "fundsCredited": typ == "WALLET_TOPUP" && decision == "APPROVE", "verified": typ == "IDENTITY_VERIFICATION" && decision == "APPROVE"}, nil
}

func (s WorkflowService) Tasks(ctx context.Context, caseID string) ([]map[string]any, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id::text,task_type,status,decision,notes,created_at::text,completed_at::text FROM workflow_tasks WHERE case_id=$1 ORDER BY created_at`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, typ, st, dec, note, created string
		var completed sql.NullString
		if err = rows.Scan(&id, &typ, &st, &dec, &note, &created, &completed); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "taskType": typ, "status": st, "decision": dec, "notes": note, "createdAt": created, "completedAt": completed.String})
	}
	return out, rows.Err()
}
