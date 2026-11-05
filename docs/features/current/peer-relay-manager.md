---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-05
source_of_truth:
  - api/openapi/openapi.yaml
  - backend/internal/service/peer_relay.go
  - backend/internal/policy/document.go
  - frontend/src/features/peer-relay/peer-relay-console.tsx
  - frontend/src/features/peer-relay/pages/
  - frontend/src/features/peer-relay/components/
  - frontend/src/components/app-sidebar.tsx
  - frontend/src/components/nav-main.tsx
  - frontend/src/components/nav-user.tsx
  - frontend/src/components/ui/sidebar.tsx
---

# Tailor Peer Relay Management

## Purpose

The console coordinates Peer Relay registration, approval, and device-to-relay grants for one Headscale instance. It manages authorization candidates; it does not force a data path or report which path a client is using.

## Roles And Identity

- Tailor authenticates through one configured external OIDC issuer using authorization code, PKCE, state, and nonce. Configure a separate IdP client for Tailor with its own client ID, secret, and callback URL.
- Tailor validates the ID token and requires a verified email that uniquely matches a current Headscale user's email. It stores only a hash of a random opaque session token in SQLite and resolves that session to the current Headscale user on each business request.
- The browser uses `GET /api/v1/auth/login`, returns through `GET /api/v1/auth/callback`, and ends the Tailor session with `POST /api/v1/auth/logout`. Logout does not sign out of the IdP. Only the health probes are public; identity headers and OIDC group claims are not accepted.
- Administrators are members of configured Headscale Policy groups (default `group:relay-admin`) or are listed in the server-side `PEER_RELAY_BOOTSTRAP_ADMINS` allowlist.
- Reviewers are members of configured groups (default `group:relay-reviewer`). Admins can review as well.
- Applicants are controlled by `PEER_RELAY_APPLICANT_MODE`: `all` (default), `groups` (configured applicant groups), or `disabled`.
- Admins can manage current Policy group mappings, applicant mode, ordinary-user self-service authorization, two reviewer delegation flags, the sync interval, and rejected-application cooldown (0–720 hours; zero disables it) in the system settings page. Headscale credentials and Policy write enablement remain server environment settings and are displayed read-only.
- Reviewer delegation can permit Relay registration management and all-device graph editing independently. A delegated Relay manager without all-device access can register only their own device.
- A regular user can view and apply changes only for devices owned by their Headscale user. Tagged devices are not self-managed by regular users.

Group membership and role grants are read from the current Headscale Policy `groups` map. The OIDC verified email is used only for login identity mapping; Tailor does not read OIDC group claims. Group and selector display is not a substitute for backend authorization.

## Headscale Synchronization

The server uses a single backend-held API key and the Headscale REST endpoints for user list, node list, and policy read/write. Startup triggers a sync, a scheduler repeats it every 60 seconds by default, and admins can request a manual sync. Admins can set the persisted interval from 10 to 3600 seconds; `HEADSCALE_SYNC_INTERVAL_SECONDS` is the initial/fallback value.

Snapshots are stored in SQLite. A failed sync preserves the most recent successful device, user, and policy snapshot and exposes sync status to the console. Devices absent from a successful snapshot are marked missing instead of being deleted, preserving assignments, relay records, and audit history; Headscale users absent from a successful snapshot are removed from the current identity mapping. A device ownership change invalidates its pending application and causes authorization checks to use the new owner.

The provider accepts the response shapes currently used by Headscale REST (`/api/v1/user`, `/api/v1/node`, `/api/v1/policy`). Before production use, validate those shapes, policy mode, selector semantics, and Peer Relay capability support against the exact deployed Headscale version.

## Relay Registration And Applications

- A relay manager can register an eligible Headscale node directly, edit a registered relay's display name, region, UDP port, public endpoint, ISP, description, and selection availability, or cancel registration.
- An eligible user can submit an application for one of their own untagged nodes. A SQLite partial unique index enforces at most one pending application per node across API processes. Admins can set a per-device cooldown after rejection.
- Review decisions are `approved` or `rejected`; a review can include a note and optional Relay metadata override. Approval creates the relay and completes the application in one SQLite transaction.
- Applicants can cancel their own pending request. A successful sync invalidates a pending request when its device is removed or no longer belongs to the applicant.
- Relay metadata includes display name, region, UDP port, public endpoint, ISP, and description. It is informational and does not configure the device or turn on its relay service.

## Grant Graph

`/editor` is a network-wide authorization canvas. Each visible Headscale node is represented once; registered Peer Relays carry a Relay role on that same node. Administrators and reviewers with all-device graph permission see the network; regular users see their own source devices and registered Relay targets. Authorization edges are derived from the current Policy and identify platform, external, inherited, unknown, and exceptional relationships. External, inherited, and unknown edges are read-only. The canvas draws each relationship from its Relay on the left to the source device on the right to show which devices are assigned to which Peer Relay; writes still update the device's Relay grants. The canvas is an authorization view, not a live network route view. Multiple grants create no priority order.

The canvas initially shows nodes with authorization links; unlinked nodes are hidden until the relationship filter is changed. Device online status remains informational. Users can arrange the layout into Relay and device columns, move nodes on a snapped grid, search by device name, user, IP, or tag, filter by Relay role, authorization links, and authorization source, zoom, pan, fit the canvas, and navigate with a minimap. A detail panel shows device status, owner, tags, Relay metadata, and related relationships; keyboard users can select a Relay target and add it. Per-user node positions and viewport are saved in that browser's local storage with a versioned key. New nodes take the next free grid cell in their role column. Assignment drafts remain in memory and prompts appear before discard or navigation with unsaved changes.

Users may edit only source devices allowed by their server-side role and per-device `canEdit` decision. Edits are reviewed as one draft and submitted through `PUT /assignments/batch`. The backend validates every source and target before changing Policy, replaces only exact grants owned by Tailor, and preserves manually managed grants and other Policy fields. If a device IP changes, its old platform-managed exact grant is removed and the current IP pair is written. Duplicate exact policy entries, unsafe selector changes, missing devices, permission failures, and stale hashes reject the complete batch. `autogroup:self` is evaluated for destination selectors using the source and destination Headscale user IDs; if either ID is unavailable, the edge stays unknown and read-only. The existing one-device graph and assignment endpoints remain available for compatibility.

## Policy Write Safety

Policy writes are disabled by default (`PEER_RELAY_POLICY_WRITES=false`). Enable them only after confirming Headscale is configured for database-backed Policy and verifying Peer Relay support in the deployed version.

For each write the backend:

1. Refreshes Headscale state and fetches the current full policy.
2. Compares the caller's base policy hash with the fresh policy; stale edits return a conflict.
3. Changes only the top-level `grants` array, retaining the other policy fields and comments outside that array.
4. Validates every changed source, target Relay, permission, IP selector, and existing grant shape before any Policy write. It refuses to create a missing `grants` array because that may change default access.
5. Merges all requested source changes into one complete Policy document, saves a pending revision and audit record, writes once, and reads it back. The revision succeeds only when the semantic hash matches. Successful revision, assignment ownership for all changed sources, and completion audit records are committed together in SQLite.

Tailor serializes policy writes within one API process and performs a final policy read immediately before each write. Headscale's Policy REST endpoint does not provide compare-and-swap; a separate writer can still change Policy between that final read and the PUT. Deployments with multiple Tailor replicas must use one policy-writing replica or an external shared write lock.

Policy input supports HuJSON comments and trailing commas. Rewriting `grants` reformats that array and does not retain comments placed inside it; comments and fields outside it are retained. Unknown selectors are displayed as read-only. The policy parser is intentionally not a general Headscale ACL editor.

Successful policy revisions keep the before/after snapshots and hashes; only the latest 30 successful revisions are retained. Admins can load one revision's snapshots on demand to preview added/removed Relay grants, affected selectors, and whether other Policy fields changed. Rollback selects only a successful revision and writes its `before` policy only if the live policy still matches the latest successful platform revision, protecting intervening external changes. Admin-only audit records expose before/after data and record pending, success, failure, or conflict outcomes for policy operations.

## API And Persistence

The API contract is `api/openapi/openapi.yaml`; routes are indexed in `docs/api/route-index.md`. Authenticated unsafe requests are limited to 60 per user per minute; OIDC entry points and public logout are limited to 20 per network peer per minute. The limiter is process-local, so multi-replica deployments need an edge or shared-store limit. SQLite stores OIDC login state, hashed Tailor sessions, Headscale snapshots, relay metadata, applications, assignment ownership, role/application settings, policy revisions, audit records, and sync state. Canvas layout is browser-local and adds no server table or API. GORM model migration runs on startup. Settings changes are audited.

## Frontend Module Boundaries

`frontend/src/app/router.tsx` declares the browser routes for the console pages, login, and not found. The Peer Relay console owns the shared page shell, permission-filtered navigation data, React Query state, and mutation orchestration; router paths select each routed page and preserve page state in the URL/history. Administrators can open the policy and settings routes, while applicants and reviewers can open the requests route. `frontend/src/components/app-sidebar.tsx` renders the responsive collapsible sidebar, `nav-main.tsx` renders page choices, and `nav-user.tsx` renders account controls. Each user-facing view remains a separate module under `frontend/src/features/peer-relay/pages/`. Feature dialogs and presentation helpers live under the feature's `components/` directory. Shared controls under `frontend/src/components/ui/` follow the configured shadcn aria-sera style and consume the global theme tokens in `frontend/src/index.css`.

## Known MVP Limits

- Single Headscale instance; no Tailscale control-plane provider.
- Authentication uses one configured OIDC issuer per deployment. OIDC group claims and IdP global logout are not supported.
- No relay health probe, automatic service enablement, route telemetry, or fastest-relay selection.
- The canvas v1 target is one Headscale instance with up to 500 devices. Dense relationships should be reviewed with representative deployment data. It does not provide a general Policy editor or shared server-side layouts.
- Existing single-device graph and assignment endpoints remain for compatibility. Peer Relay list APIs still return complete arrays; directory pagination is in the browser.
- Compatibility must be verified against the deployed Headscale build before enabling writes.
