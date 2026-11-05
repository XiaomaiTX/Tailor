---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/cmd/api/main.go
  - backend/internal/router/router.go
  - frontend/src/main.tsx
  - api/openapi/openapi.yaml
---

# Architecture Overview

## Scope

Describes the current Peer Relay Manager structure. OpenAPI is the shared API contract; code is the implementation source of truth.

## Current Implementation

- Product: single-instance Headscale Peer Relay registration, approval, and grant management console.
- Backend: Go 1.27.3, Gin, GORM/SQLite, Zap; one HTTP process with a background Headscale synchronization scheduler.
- Frontend: React 19, Vite, React Query, React Router, Tailwind, shadcn-style components, Zustand, and i18n.
- API contract: OpenAPI 3.0.3. React API client and models are generated from the contract. The generated Go strict handler remains for health and retained Example routes; a dedicated Gin adapter binds Peer Relay operations.
- Authentication: Tailor uses external OIDC and maps a verified email to one current Headscale user, then authorizes business requests through a server-side SQLite session.
- Persistence: SQLite for authentication states and sessions, Headscale snapshots, relay registrations, applications, assignment ownership, policy revisions, audit, and sync status.
- Deployment example: Compose runs backend plus frontend Nginx; production configures a dedicated Tailor OIDC client, uses HTTPS, and keeps the backend on a private network.

## Subsystems

- **Headscale provider**: REST access and decoding in `backend/internal/headscale/`.
- **Policy model**: HuJSON parsing, grant classification, safe grant replacement, and semantic hashes in `backend/internal/policy/`.
- **Business service**: authorization, registration/review lifecycle, graph projection, policy commits, rollback, audit, and sync in `backend/internal/service/peer_relay.go`.
- **Persistence**: GORM models and repository in `backend/internal/model/peer_relay.go` and `backend/internal/repository/peer_relay.go`.
- **Frontend console**: `frontend/src/App.tsx` delegates to `frontend/src/features/peer-relay/peer-relay-console.tsx`; each dashboard, relay directory, application, graph editor, policy, and settings view lives in its own feature page module.

## Key Entry Files

- Backend entry: [backend/cmd/api/main.go](../../backend/cmd/api/main.go)
- Route assembly: [backend/internal/router/router.go](../../backend/internal/router/router.go)
- Peer Relay HTTP adapter: [backend/internal/handler/peer_relay.go](../../backend/internal/handler/peer_relay.go)
- Headscale provider: [backend/internal/headscale/client.go](../../backend/internal/headscale/client.go)
- Policy document adapter: [backend/internal/policy/document.go](../../backend/internal/policy/document.go)
- Frontend SPA entry: [frontend/src/App.tsx](../../frontend/src/App.tsx)
- Peer Relay console: [frontend/src/features/peer-relay/peer-relay-console.tsx](../../frontend/src/features/peer-relay/peer-relay-console.tsx)
- Contract: [api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)

## Invariants

- Business rules stay in service, never only in the UI or HTTP adapter.
- Backend authorization checks identity and device ownership; client-controlled owner values are not trusted.
- Policy commits re-read the live policy, use a base hash, mutate only grants, and verify the write by reading it back.
- API keys remain backend-side, policy writes default off, and unknown grants are retained.
- See `module-map.md`, `auth-and-permissions.md`, and `docs/features/current/peer-relay-manager.md` for details.
