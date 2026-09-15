-- NusaMedia Control Center: GUI-first control plane and explicit Admin Root authority.
CREATE TABLE IF NOT EXISTS admin_role_assignments(
  user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  role_name text NOT NULL,
  permissions jsonb NOT NULL DEFAULT '[]'::jsonb,
  granted_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS control_plane_settings(
  key text PRIMARY KEY,
  value text NOT NULL DEFAULT '',
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deployment_jobs(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  target text NOT NULL,
  environment text NOT NULL,
  version text NOT NULL,
  provider text NOT NULL,
  status text NOT NULL DEFAULT 'PENDING',
  message text NOT NULL DEFAULT '',
  requested_by uuid REFERENCES users(id),
  approved_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mobile_builds(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  platform text NOT NULL,
  version text NOT NULL,
  build_number text NOT NULL,
  environment text NOT NULL,
  provider text NOT NULL DEFAULT 'EAS',
  status text NOT NULL DEFAULT 'PENDING',
  store_track text NOT NULL DEFAULT 'INTERNAL',
  message text NOT NULL DEFAULT '',
  requested_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_role_assignments_role ON admin_role_assignments(role_name);
CREATE INDEX IF NOT EXISTS idx_deployment_jobs_created ON deployment_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mobile_builds_created ON mobile_builds(created_at DESC);

-- Default role catalog. Permissions are intentionally conservative; ADMIN_ROOT is enforced in code.
INSERT INTO admin_roles(name, permissions) VALUES
('ADMIN_ROOT', '["*"]'::jsonb),
('ADMIN_LEVEL_2', '["dashboard.read","users.read","content.moderate","verification.review"]'::jsonb),
('DEVELOPER', '["dashboard.read","deployment.read","mobile.read"]'::jsonb),
('RELEASE_MANAGER', '["dashboard.read","deployment.read","deployment.request","mobile.read","mobile.build","mobile.submit"]'::jsonb),
('MODERATOR', '["dashboard.read","content.moderate","reports.review"]'::jsonb)
ON CONFLICT(name) DO NOTHING;
