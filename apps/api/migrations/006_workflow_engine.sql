-- NusaMedia Workflow Engine: controlled human-in-the-loop operations.
-- Financial and identity-sensitive actions never mutate protected state merely because a user submitted a request.
CREATE TABLE IF NOT EXISTS workflow_cases(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  case_type text NOT NULL,
  subject_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  source_ref text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'OPEN',
  priority text NOT NULL DEFAULT 'NORMAL',
  amount bigint,
  currency text NOT NULL DEFAULT 'IDR',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  assigned_department_id uuid REFERENCES organization_departments(id) ON DELETE SET NULL,
  assigned_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz
);

CREATE TABLE IF NOT EXISTS workflow_tasks(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id uuid NOT NULL REFERENCES workflow_cases(id) ON DELETE CASCADE,
  task_type text NOT NULL,
  department_id uuid REFERENCES organization_departments(id) ON DELETE SET NULL,
  assignee_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  status text NOT NULL DEFAULT 'PENDING',
  decision text NOT NULL DEFAULT '',
  notes text NOT NULL DEFAULT '',
  evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);

CREATE TABLE IF NOT EXISTS workflow_events(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id uuid NOT NULL REFERENCES workflow_cases(id) ON DELETE CASCADE,
  actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
  event_type text NOT NULL,
  from_status text NOT NULL DEFAULT '',
  to_status text NOT NULL DEFAULT '',
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workflow_approvals(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id uuid NOT NULL REFERENCES workflow_cases(id) ON DELETE CASCADE,
  task_id uuid REFERENCES workflow_tasks(id) ON DELETE SET NULL,
  requested_by uuid REFERENCES users(id) ON DELETE SET NULL,
  approved_by uuid REFERENCES users(id) ON DELETE SET NULL,
  status text NOT NULL DEFAULT 'PENDING',
  reason text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  decided_at timestamptz
);

CREATE TABLE IF NOT EXISTS topup_requests(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id uuid UNIQUE REFERENCES workflow_cases(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  amount bigint NOT NULL CHECK(amount > 0),
  payment_method text NOT NULL,
  provider_reference text NOT NULL DEFAULT '',
  evidence_url text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'PENDING_REVIEW',
  review_note text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE verification_requests ADD COLUMN IF NOT EXISTS workflow_case_id uuid REFERENCES workflow_cases(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workflow_cases_status_created ON workflow_cases(status,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_cases_department_status ON workflow_cases(assigned_department_id,status);
CREATE INDEX IF NOT EXISTS idx_workflow_cases_subject ON workflow_cases(subject_user_id,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_case ON workflow_tasks(case_id,status);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_department ON workflow_tasks(department_id,status,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_events_case_created ON workflow_events(case_id,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_approvals_case_status ON workflow_approvals(case_id,status);
CREATE INDEX IF NOT EXISTS idx_topup_requests_status_created ON topup_requests(status,created_at DESC);

-- Expand the approved role catalog with operational separation.
INSERT INTO admin_roles(name, permissions) VALUES
('FINANCE_MANAGER','["dashboard.read","finance.review","finance.approve","workspace.read","workspace.notify"]'::jsonb),
('FINANCE_REVIEWER','["dashboard.read","finance.review","workspace.read","workspace.notify"]'::jsonb),
('VERIFICATION_MANAGER','["dashboard.read","verification.review","verification.approve","workspace.read","workspace.notify"]'::jsonb),
('VERIFICATION_REVIEWER','["dashboard.read","verification.review","workspace.read","workspace.notify"]'::jsonb),
('SECURITY_ADMIN','["dashboard.read","security.review","workspace.read","workspace.notify"]'::jsonb),
('OPERATIONS_MANAGER','["dashboard.read","operations.review","workspace.read","workspace.notify"]'::jsonb)
ON CONFLICT(name) DO NOTHING;
