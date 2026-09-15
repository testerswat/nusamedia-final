-- NusaMedia Organization & Workspace: department ownership, scoped access and stakeholder communication.
CREATE TABLE IF NOT EXISTS organization_departments(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text UNIQUE NOT NULL,
  name text UNIQUE NOT NULL,
  description text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'ACTIVE',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspaces(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  department_id uuid REFERENCES organization_departments(id) ON DELETE SET NULL,
  name text NOT NULL,
  slug text UNIQUE NOT NULL,
  description text NOT NULL DEFAULT '',
  visibility text NOT NULL DEFAULT 'PRIVATE',
  status text NOT NULL DEFAULT 'ACTIVE',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspace_members(
  workspace_id uuid REFERENCES workspaces(id) ON DELETE CASCADE,
  user_id uuid REFERENCES users(id) ON DELETE CASCADE,
  membership_role text NOT NULL DEFAULT 'MEMBER',
  status text NOT NULL DEFAULT 'ACTIVE',
  granted_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(workspace_id,user_id)
);

CREATE TABLE IF NOT EXISTS workspace_channels(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name text NOT NULL,
  channel_type text NOT NULL DEFAULT 'GENERAL',
  visibility text NOT NULL DEFAULT 'PRIVATE',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(workspace_id,name)
);

CREATE TABLE IF NOT EXISTS workspace_messages(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  channel_id uuid NOT NULL REFERENCES workspace_channels(id) ON DELETE CASCADE,
  sender_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  body text NOT NULL,
  priority text NOT NULL DEFAULT 'NORMAL',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspace_notifications(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  channel_id uuid REFERENCES workspace_channels(id) ON DELETE CASCADE,
  sender_id uuid REFERENCES users(id) ON DELETE SET NULL,
  recipient_user_id uuid REFERENCES users(id) ON DELETE CASCADE,
  title text NOT NULL,
  body text NOT NULL,
  event_type text NOT NULL DEFAULT 'WORKSPACE_MESSAGE',
  priority text NOT NULL DEFAULT 'NORMAL',
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_workspaces_department ON workspaces(department_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id,status);
CREATE INDEX IF NOT EXISTS idx_workspace_channels_workspace ON workspace_channels(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_messages_channel_created ON workspace_messages(channel_id,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workspace_notifications_recipient_created ON workspace_notifications(recipient_user_id,created_at DESC);

INSERT INTO organization_departments(code,name,description) VALUES
('EXEC','Executive','Strategic direction, governance and company-wide decisions'),
('ENG','Engineering','Product engineering, platform, mobile and infrastructure'),
('SEC','Security','Security, privacy, identity, risk and incident response'),
('FIN','Finance','Finance, accounting, payments and financial controls'),
('OPS','Operations','Platform operations, support and service management'),
('PROD','Product','Product management, research, UX and roadmap'),
('GROWTH','Growth','Marketing, partnerships, community and growth'),
('LEGAL','Legal & Compliance','Legal, policy, compliance and regulatory affairs'),
('TRUST','Trust & Safety','Moderation, reports, verification and user safety')
ON CONFLICT(code) DO NOTHING;

INSERT INTO workspaces(department_id,name,slug,description) 
SELECT d.id,'Executive Office','executive-office','Company-wide executive workspace' FROM organization_departments d WHERE d.code='EXEC'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Engineering','engineering','Engineering and platform delivery workspace' FROM organization_departments d WHERE d.code='ENG'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Security & Privacy','security-privacy','Security, privacy and incident workspace' FROM organization_departments d WHERE d.code='SEC'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Finance','finance','Finance and payment operations workspace' FROM organization_departments d WHERE d.code='FIN'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Operations','operations','Operations and service management workspace' FROM organization_departments d WHERE d.code='OPS'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Product','product','Product, UX and roadmap workspace' FROM organization_departments d WHERE d.code='PROD'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Growth & Partnerships','growth-partnerships','Growth, partnerships and community workspace' FROM organization_departments d WHERE d.code='GROWTH'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Legal & Compliance','legal-compliance','Legal and compliance workspace' FROM organization_departments d WHERE d.code='LEGAL'
ON CONFLICT(slug) DO NOTHING;
INSERT INTO workspaces(department_id,name,slug,description)
SELECT d.id,'Trust & Safety','trust-safety','Moderation, verification and safety workspace' FROM organization_departments d WHERE d.code='TRUST'
ON CONFLICT(slug) DO NOTHING;

INSERT INTO workspace_channels(workspace_id,name,channel_type,visibility)
SELECT w.id,'General','GENERAL','PRIVATE' FROM workspaces w
ON CONFLICT(workspace_id,name) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_department_assignments(
  user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  department_id uuid NOT NULL REFERENCES organization_departments(id) ON DELETE RESTRICT,
  job_title text NOT NULL DEFAULT '',
  granted_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_user_department_department ON user_department_assignments(department_id);
