# Production Hardening

## Security
- JWT secret only through secret manager/environment.
- No production credentials in source or ZIP.
- Security response headers enabled.
- Request context bounded by server timeout.
- Admin endpoints require admin role.
- Account type cannot be selected during registration.
- Passwords use bcrypt; production should add breached-password policy and MFA for privileged roles.
- Production CORS must be allowlisted per deployed origin; wildcard CORS is for development foundation only.

## Scale
- PostgreSQL is source of truth.
- Redis/Valkey is reserved for cache, rate limits, realtime fan-out and job coordination.
- Object storage + CDN is required for production media.
- Worker processes are required for media transcoding, notifications, moderation, AI jobs and commerce reconciliation.

## Observability
Use OpenTelemetry-compatible tracing/metrics, structured JSON logs, error tracking, database metrics, queue lag and audit event dashboards.

## Release gate
A release is not production-ready until:
1. `go test ./...` passes in a networked build environment.
2. Web/Admin `npm ci && npm run typecheck && npm run build` pass.
3. Integration/E2E suites cover auth, social, discover, verification, commerce, wallet, admin and failure paths.
4. Load tests cover feed/search/checkout and websocket fan-out.
5. Security scans cover dependencies, containers, JWT/session handling and API authorization.
6. Provider credentials are configured and tested in staging.

## Organization & stakeholder controls

- Departments and workspaces are first-class data domains.
- Workspace membership is distinct from global admin role.
- Workspace notifications fan out only to active members of the selected workspace.
- Admin Root controls privileged membership/channel changes.
- Root-only endpoints revalidate the current database role, not only the JWT role claim.
- Admin Root bootstrap uses a PostgreSQL advisory transaction lock to prevent concurrent first-root creation.
