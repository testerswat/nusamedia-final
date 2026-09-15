# NusaMedia Control Center — GUI-first control plane

NusaMedia is operated through the Admin Center/Control Center rather than requiring an operator to type terminal commands.

## Authority model

- `ADMIN_ROOT` is the only role allowed to mutate the control plane.
- Admin Root is established by the one-time installer bootstrap/recovery procedure.
- `ADMIN_LEVEL_2`, `DEVELOPER`, `RELEASE_MANAGER`, `MODERATOR`, and `STAKEHOLDER_FULL` can be granted by Admin Root.
- Lower roles can use the areas allowed to them, but cannot create/promote another Admin Root or mutate provider integrations, infrastructure/release requests, or root authority.
- Root mutation endpoints re-check the current database role on every request, so an old JWT cannot retain Root authority after demotion.

## GUI operations

The Control Center provides GUI flows for:

- platform dashboard and health
- deployment request/history
- Android/iOS build request/history
- provider integrations
- administrative role assignment/revocation
- infrastructure configuration surface
- audit/security surface

The operator uses forms, buttons, selectors, status indicators, approval gates, and history views. No command syntax is part of the normal workflow.

## Mobile release

Android/iOS release orchestration is provider-backed. The Control Center records and authorizes the requested build/release, while a configured provider such as EAS/App Store Connect/Google Play performs the actual cloud build and store submission.

A provider must be connected before production execution is enabled. Credentials are never intended to be placed in application source code.

## Important implementation boundary

This repository now contains the GUI and backend control-plane contract. Provider-specific execution adapters are intentionally separate from the GUI so credentials and vendor APIs are not hard-coded. A deployment/build request is persisted with an explicit status and provider. The next implementation step for a live provider is to connect the corresponding adapter/secret store and worker; the GUI does not pretend that an unconnected provider has completed a deployment.

## Operational Workflow Control
The Control Center now includes a human-in-the-loop Operations & Workflows view. Sensitive requests become cases with tasks and approval gates. Finance and Trust & Safety actions are not direct toggles: reviewers record a decision, the case transitions to a manager approval state, and only final approval performs the protected mutation. The API enforces role and separation-of-duties checks server-side.
