# NusaMedia Final — Architecture

```text
                 ┌───────────────────────────────┐
                 │        NusaMedia Clients      │
                 │ Web / Mobile / Admin Center   │
                 └──────────────┬────────────────┘
                                │ HTTPS / JSON
                 ┌──────────────▼────────────────┐
                 │       Go API / Control Plane  │
                 │ auth • RBAC • validation      │
                 │ services • audit • health     │
                 └───────┬───────────┬───────────┘
                         │           │
                 ┌───────▼─────┐ ┌───▼────────────┐
                 │ PostgreSQL  │ │ Redis / Valkey  │
                 │ source of   │ │ cache/realtime  │
                 │ truth       │ │ queue fan-out   │
                 └───────┬─────┘ └───────┬────────┘
                         │               │
                 ┌───────▼───────────────▼────────┐
                 │ Object Storage + CDN + Workers  │
                 │ media • notifications • AI     │
                 │ moderation • commerce jobs     │
                 └─────────────────────────────────┘
```

The source intentionally keeps provider adapters separate from core business rules. Production secrets are environment/secret-manager concerns.

## Organization, Workspace & Governance

NusaMedia uses an organization layer above operational administration. Departments define business ownership (Executive, Engineering, Security, Finance, Operations, Product, Growth, Legal/Compliance, Trust & Safety). Each department may own one or more private workspaces.

Workspace membership is separate from platform role. A person may be a Developer in the platform and only a member of the Engineering workspace. Membership and channel scope must be explicitly granted by Admin Root for privileged workspace administration.

Workspace messages create scoped stakeholder notifications for active members. This provides a company communication plane without granting recipients platform-wide privileges.

Admin Root remains the highest authority. ADMIN_LEVEL_2, developers, managers, finance, release managers, moderators and stakeholders cannot promote themselves, create Admin Root, or change Root-only infrastructure/security/release controls. Current administrative role is revalidated against the database on protected Admin Center requests so revocation takes effect even when an older JWT still exists.
