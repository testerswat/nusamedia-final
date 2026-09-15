# NusaMedia — Final Full-Stack Platform

Paket source **asli** NusaMedia. Tidak menyertakan export/deployment AppDeploy, snapshot deployment, `node_modules`, `dist`, atau credential produksi.

## Arsitektur
- `apps/web` — React 19.3 + Vite 8.1 web application, mobile-first.
- `apps/mobile` — Expo/React Native mobile client foundation yang memakai kontrak API yang sama.
- `apps/admin` — Admin Center/control plane.
- `apps/api` — Go 1.26 HTTP API, JWT, PostgreSQL, security middleware, service/use-case layer.
- `apps/api/migrations` — PostgreSQL 18 schema dan platform data.
- `packages/contracts` — OpenAPI contract.
- `docs` — architecture, UI, completion dan operational contracts.

## Prinsip produk
1. Satu produk: **NusaMedia Final**. Tidak ada v2/v3 sebagai nama produk.
2. Registrasi tidak meminta account type.
3. Account type ditentukan melalui proses verifikasi setelah akun dibuat.
4. Setiap fitur harus melewati UI → route → API contract → auth/RBAC → service → persistence/provider → output → loading/empty/error → test.
5. Credential produksi hanya melalui secret manager/environment; tidak pernah disimpan di source.

## Modern technology baseline
Source menargetkan React 19.3 dan Vite 8.1; React 19.3 adalah rilis stabil terbaru pada September 2026 dan Vite 8 memakai Rolldown sebagai bundler. Go ditargetkan 1.26. PostgreSQL ditargetkan major 18, dengan minor production mengikuti current supported release. 

## Jalankan
`docker compose up --build`

Web: `http://localhost:8080`
Admin: `http://localhost:8080/admin`
API health: `http://localhost:8080/api/health`

## Production requirements
- PostgreSQL managed/HA atau cluster yang sesuai kebutuhan.
- Redis/Valkey untuk cache, rate limit dan realtime fan-out.
- Object storage S3-compatible + CDN untuk foto/video.
- Secret manager untuk JWT, database, payment, maps, email, push dan AI provider.
- Worker queue untuk media processing, notification, moderation dan order lifecycle.
- Observability: structured logs, metrics, traces, error tracking dan audit events.
