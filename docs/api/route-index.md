---
status: active
doc_type: api
owner: engineering
last_reviewed: 2026-11-05
source_of_truth:
  - api/openapi/openapi.yaml
  - backend/internal/router/router.go
  - backend/internal/handler/peer_relay.go
---

# Route Index

## Scope

Stable HTTP routes under `/api/v1`. Health probes and OIDC entry/callback routes are public; business routes require a valid OIDC-backed Tailor session. See `docs/architecture/auth-and-permissions.md`.

## Interface Table

| Method | Route | Purpose | Auth / Role | Consumer |
| --- | --- | --- | --- | --- |
| GET | `/health/live` | Process liveness | Public | Compose, monitoring |
| GET | `/health/ready` | SQLite readiness | Public | Monitoring |
| GET | `/auth/login` | Start OIDC authorization-code login | Public; sets short-lived state cookie | Browser |
| GET | `/auth/callback` | Validate OIDC response and establish session | Public callback; requires matching state cookie | IdP |
| POST | `/auth/logout` | Revoke local Tailor session | Public and idempotent; validates Origin | Console |
| GET | `/relay/me` | Current user and role capabilities | Tailor session; current Headscale user | Console |
| GET | `/relay/dashboard` | Counts, sync state, policy hash, capabilities | Tailor session; current Headscale user | Console |
| GET | `/settings` | Read role, application, and server runtime settings | Admin | System settings |
| PUT | `/settings` | Update Policy group roles, application/self-service policy, reviewer delegation, sync interval, and rejection cooldown | Admin; groups must exist in the current Policy and retain an admin group | System settings |
| GET | `/devices?search=` | List visible devices | Tailor session; admin sees all, other users see own | Console |
| GET | `/relays?includeAll=` | List selectable relays; admin may include unavailable entries | Tailor session; `includeAll=true` is admin-only | Console |
| POST | `/relays` | Directly register a relay | Admin | Console |
| PUT | `/relays/{nodeId}` | Update relay metadata/selectability | Admin | Console |
| DELETE | `/relays/{nodeId}` | Cancel a relay registration | Admin; conflicts if references remain | Console |
| GET | `/applications?status=` | List requests | Tailor session; reviewer sees all, others see own | Console |
| POST | `/applications` | Apply to register an owned device | Applicant; owned untagged device | Console |
| POST | `/applications/{id}/cancel` | Cancel own pending request | Applicant; request owner | Console |
| POST | `/applications/{id}/review` | Approve or reject a request | Reviewer | Console |
| GET | `/graph/all` | Read the visible network authorization canvas and one Policy baseline | Tailor session; admin/reviewer sees permitted network scope, regular user sees own sources and registered Relay targets | Authorization canvas |
| GET | `/graph?sourceNodeId=` | View source device and relay authorization edges | Tailor session; admin or source owner | Single-device compatibility API |
| PUT | `/assignments` | Merge and commit graph edits to Headscale Policy | Admin or source owner; Policy writes enabled | Single-device compatibility API |
| PUT | `/assignments/batch` | Validate changed source devices together and commit one merged Policy update | Admin or source owner per changed device; Policy writes enabled | Authorization canvas |
| GET | `/policy/selector-rules` | List Tailor-owned user/group selector rules with current matching devices | Admin | Policy management |
| POST | `/policy/selector-rules/preview` | Preview one user/group selector to one registered Relay and return policy/scope hashes | Admin | Policy management |
| POST | `/policy/selector-rules` | Write one user/group selector rule without expanding the source selector | Admin; Policy writes enabled; preview hashes must match | Policy management |
| DELETE | `/policy/selector-rules/{id}?basePolicyHash=` | Remove one exact Tailor-owned selector grant or clear its orphaned ownership row | Admin; Policy writes enabled when a Policy grant must be removed | Policy management |
| POST | `/sync` | Trigger immediate Headscale sync | Admin | Console |
| GET | `/policy/revisions` | List recent policy revisions | Admin | Console |
| GET | `/policy/revisions/{id}` | Read full before/after policy snapshots | Admin | Console |
| POST | `/policy/revisions/{id}/rollback` | Restore a revision's before-policy | Admin; live hash must match latest platform revision | Console |
| GET | `/audit` | List audit records | Admin | Console |
| GET/POST | `/examples` | Retained starter CRUD example | Tailor session | Generated API example |
| GET/PUT/DELETE | `/examples/{id}` | Retained starter CRUD example | Tailor session | Generated API example |

## Contract Details

- `api/openapi/openapi.yaml` is the cross-client contract and security declaration.
- Responses use JSON; errors use `application/problem+json` with `type`, `title`, `status`, optional `detail`, and `instance`.
- Peer Relay request decoding is strict, rejects unknown fields, and caps JSON bodies at 1 MiB. The generated strict Go handler remains responsible for health and retained examples; Peer Relay routes are bound in `internal/handler/peer_relay.go` and use the service layer.
- `/graph/all` returns one device node per visible Headscale device, marks registered Peer Relay roles on those nodes, and scopes source nodes by the caller's server-enforced permissions. `/assignments/batch` accepts only changed sources with their complete desired Relay ID sets and one shared Policy hash; the service validates the whole request before applying one Policy document update.
- `GET /policy/selector-rules` returns Tailor-owned user/group rules and their current source-device matches. Preview returns the exact registered Relay IPv4 target, current source set, Policy hash, and scope hash; create must submit both hashes so changes to Policy or the matched device set require a new preview. Selector writes preserve the source selector and are administrator-only.
- Authorization canvas node positions and viewport remain in browser local storage, isolated by Tailor username and layout version. Assignment drafts are in memory only.
- `page`/`pageSize` pagination applies to retained Example endpoints. Peer Relay collection responses are currently bounded by the data returned and are not paginated.
- Times use ISO 8601 `date-time`.
- Conflict responses identify stale policy hashes, ambiguous grants, or invalid state transitions. `503` covers unavailable OIDC/Headscale dependencies, disabled Policy writes, or unavailable Headscale.

## How To Maintain

- Update `api/openapi/openapi.yaml` first, then implementation/client and this route table.
- Peer Relay operation types are generated into the React client with `orval`; the Go generator includes only `health` and `examples` tags because Peer Relay routes use a dedicated Gin handler adapter.
- Peer Relay collection APIs return complete arrays; the console applies browser-side pagination. Mutating methods, OIDC entry points, and public logout can return `429` with `Retry-After`.
- Run OpenAPI lint and generation checks when the local toolchain is available. Record environment blocks and generated drift in the delivery notes.
