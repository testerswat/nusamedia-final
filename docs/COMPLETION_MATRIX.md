# NusaMedia Completion Matrix

Every product capability must pass this contract before release:

1. UI layout exists
2. Frontend route exists
3. Typed API client exists
4. API endpoint exists
5. Auth/RBAC is enforced
6. Service/use-case exists
7. Repository/database persistence exists
8. Success response is rendered in UI
9. Loading/empty/error states exist
10. Audit/event side effects exist where applicable
11. E2E/integration test exists

## Core Social
Feed, create post, post detail, comments, likes, saves, shares, follow, profile, stories, reels, discover, search, notifications, messages.

## Marketplace
Shop, products, categories, cart, checkout, payment adapter, shipping adapter, orders, reviews, seller operations.

## Platform
Verification, moderation, reports, admin root, RBAC, integrations, webhooks, feature flags, audit/security, installer, health/observability.

## Clients
Responsive web, Admin Center, mobile foundation/application using the same API contracts.
