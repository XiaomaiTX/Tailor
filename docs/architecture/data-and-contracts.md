---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/internal/model/
  - api/openapi/openapi.yaml
  - backend/internal/problem/problem.go
---

# Data And Contracts

## Scope

Describes SQLite persistence, shared API contracts, and the Headscale integration boundary.

## Data Stores

- **SQLite** is the only local persistent store; `DATABASE_PATH` selects the file.
- GORM models are auto-migrated at startup. SQLite stores short-lived OIDC login state, hashed Tailor session tokens, Headscale user/device snapshots, relay metadata, applications, source-to-relay assignment ownership, role/application settings, policy revisions, audit logs, and sync state.
- Successful Headscale syncs replace the current user snapshot. User records absent from that snapshot are removed so removed users cannot continue logging in through stale email mappings.
- Device snapshots are marked missing when absent from a successful Headscale sync; rows are retained so historical ownership and references remain inspectable.
- Policy revision rows contain full before/after policy snapshots and semantic hashes. The service retains the latest 30 successful revisions.
- The singleton settings row stores configurable Policy groups, applicant mode, self-service authorization, reviewer delegation, sync interval, and rejected-application cooldown. Headscale credentials and the Policy write switch remain backend environment configuration; the environment sync interval is the initial/fallback value.
- A partial unique SQLite index allows at most one pending application for a node, including when more than one API process submits concurrently.

## Shared Contracts

- `api/openapi/openapi.yaml` is the cross-client contract.
- React client and model types live under `frontend/src/api/generated/` and are produced by Orval.
- The Go OpenAPI generator is limited to the health/example tags. Peer Relay REST routes use `backend/internal/handler/peer_relay.go`, a strict JSON Gin adapter, and the same OpenAPI contract.
- Errors are RFC 7807-style `application/problem+json` values from `internal/problem`.

## External Contract

- The Headscale provider is `backend/internal/headscale/client.go` and uses the configured base URL, API key, and REST API endpoints for users, nodes, and policy.
- Headscale credentials remain in backend configuration. Policy writes require database Policy mode and are disabled by default.
- The policy parser accepts HuJSON comments and trailing commas and changes only the `grants` value. It preserves the rest of the source policy, but comments inside a rewritten `grants` array are not preserved.
- The product does not own general ACL policy editing and refuses unsafe or ambiguous changes.

## Invariants

- Update OpenAPI before changing a cross-boundary request or response.
- Preserve existing policy content outside the platform-managed grant edits.
- Never expose Headscale credentials or full policy snapshots to frontend responses.
- See `docs/features/current/peer-relay-manager.md` for data lifecycle and write behavior.
