---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-05
source_of_truth:
  - api/openapi/openapi.yaml
  - backend/internal/service/peer_relay.go
  - backend/internal/policy/document.go
  - frontend/src/app/router.tsx
  - frontend/src/App.tsx
  - frontend/src/features/peer-relay/queries.ts
  - frontend/src/features/peer-relay/pages/
  - frontend/src/features/peer-relay/components/
  - frontend/src/components/app-sidebar.tsx
  - frontend/src/components/nav-main.tsx
  - frontend/src/components/nav-user.tsx
  - frontend/src/components/ui/sidebar.tsx
---

# Tailor Peer Relay Management

## Purpose

Tailor manages Peer Relay registration, application review, and device-to-Relay grants for one Headscale instance. The console reads Headscale Policy to show current grants and can update Tailor-managed Relay grants when policy writes are enabled. It does not start Relay services or report live traffic paths.

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

The server uses a single backend-held API key and the Headscale REST endpoints for user list, node list, and policy read/write. Startup triggers a sync, a scheduler repeats it every 60 seconds by default, and admins can request a manual sync. Admins can set the persisted interval from 10 to 3600 seconds; `HEADSCALE_SYNC_INTERVAL_SECONDS` is the initial/fallback value. Tailor does not infer compatibility from a Tailscale feature list or currently expose the Headscale server version; deployments must verify selector behavior and the connected client's `peer-relay` status against their exact Headscale build.

Snapshots are stored in SQLite. A failed sync preserves the most recent successful device, user, and policy snapshot and exposes sync status to the console. Devices absent from a successful snapshot are marked missing instead of being deleted, preserving assignments, relay records, and audit history; Headscale users absent from a successful snapshot are removed from the current identity mapping. A device ownership change invalidates its pending application and causes authorization checks to use the new owner.

The provider accepts the response shapes currently used by Headscale REST (`/api/v1/user`, `/api/v1/node`, `/api/v1/policy`). Before production use, validate those shapes, policy mode, selector semantics, and Peer Relay capability support against the exact deployed Headscale version.

## Headscale Selector Compatibility

This matrix describes Tailor's current evaluation and write boundary against Headscale's documented Policy model. The [Headscale Policy reference](https://headscale.net/development/ref/policy/) documents the selector behavior and limitations below; the [Headscale changelog](https://github.com/juanfont/headscale/blob/main/CHANGELOG.md) records the `cap/relay` capability. Those sources do not certify a particular deployment build. The repository has no deployed Headscale version or client connection state, so deployment validation must still confirm the exact server build and that a client reports `peer-relay` after policy application.

| Policy form | Headscale documentation | Tailor current evaluation | Tailor write support |
| --- | --- | --- | --- |
| `tailscale.com/cap/relay` | Headscale changelog records the Relay capability; runtime support still depends on the deployed build and client | Parses safe Relay grants and verifies Policy read-back | Existing per-device writes and the first selector-rule flow can write it; client `peer-relay` status is not inferred |
| User selector, `group:`, nested groups | Supported Policy selectors | Matches synchronized user/email and OR group membership; unresolved nested branches retain uncertainty | Admin-only, one current user or fully resolvable group to one registered Relay IPv4 |
| `tag:`, hosts alias, hostname, IP/CIDR | Supported selector forms | Matches current synchronized tags, aliases, names, and addresses | Read-only; no selector-preserving writes yet |
| Multiple `src` or `dst` selectors | Selector arrays are sets of matches | Expands current matches per source and Relay without merging unrelated pairs | Read-only; no list writes |
| `autogroup:member`, `tagged`, `self` | Supported subset; `self` is destination-only and excludes tagged devices | Evaluates member/tagged and destination `self` when user IDs are known | Read-only |
| `autogroup:internet` | Destination for exit-node access | Recognized as exit-node scope, but current exit-node route state is not synchronized | Read-only and uncertain |
| `autogroup:danger-all` | Source-only; includes all IP addresses | Recognized as broad source scope and shown with a warning | Read-only |
| IP sets and device posture | Headscale documents these as unsupported | Not evaluated; affected relations remain unknown | Read-only |
| `tagOwners` | Tag assignment metadata, not a Grant selector | Not treated as a Peer Relay authorization | Not writable here |
| `via`, extra capabilities, or additional grant fields | Depends on the exact Headscale feature and version | Not fully evaluated; shown as uncertain | Read-only |

The tagged-device distinction also follows [Tailscale tag identity rules](https://tailscale.com/docs/features/tags), which state that a device has a tag identity or a user account. Runtime Peer Relay behavior remains version-specific and must be checked with the deployed Headscale build and a connected Tailscale client.

## Relay Registration And Applications

- A relay manager can register an eligible Headscale node directly, edit a registered relay's display name, region, UDP port, public endpoint, ISP, description, and selection availability, or cancel registration.
- An eligible user can submit an application for one of their own untagged nodes. A SQLite partial unique index enforces at most one pending application per node across API processes. Admins can set a per-device cooldown after rejection.
- Review decisions are `approved` or `rejected`; a review can include a note and optional Relay metadata override. Approval creates the relay and completes the application in one SQLite transaction.
- Applicants can cancel their own pending request. A successful sync invalidates a pending request when its device is removed or no longer belongs to the applicant.
- Relay metadata includes display name, region, UDP port, public endpoint, ISP, and description. It is informational and does not configure the device or turn on its relay service.

## Grant Graph

`/editor` is a network-wide authorization canvas. Each visible Headscale node is represented once; registered Peer Relays carry a Relay role on that same node. Administrators and reviewers with all-device graph permission see the network; regular users see their own source devices and registered Relay targets. Authorization edges are derived from the current Policy and identify platform, external, inherited, unknown, and exceptional relationships. Each evaluated edge includes the matching `src` and `dst` selectors plus a content-derived grant ID. That ID is stable when the grant is only reformatted, but it is not a native Headscale grant identifier. External, inherited, unknown, and exceptional edges are read-only. A rule that Tailor cannot fully evaluate is shown as unknown instead of confirmed authorization. The canvas draws each relationship from its Relay on the left to the source device on the right to show which devices are assigned to which Peer Relay; writes still update the device's Relay grants. The canvas is an authorization view, not a live network route view. Multiple grants create no priority order.

The Policy page also manages selector rules separately from per-device `RelayAssignment` rows. Administrators can write one current Headscale user selector (for example `chichu@`) or one existing `group:` selector to one registered, selectable Relay's current Tailscale IPv4 address. A preview lists current source devices and the target Relay, and the submitted policy and scope hashes reject changes between preview and write. The grant retains its original selector, so new devices and group membership changes are evaluated from the latest synced snapshot. Selector rules are audited and have their own ownership rows. Tags, hosts aliases, CIDRs, lists, autogroups, and other complex selector forms remain read-only for writes.

The canvas initially shows nodes with authorization links; unlinked nodes are hidden until the relationship filter is changed. Device online status remains informational. Users can arrange the layout into Relay and device columns, move nodes on a snapped grid, search by device name, user, IP, or tag, filter by Relay role, authorization links, and authorization source, zoom, pan, fit the canvas, and navigate with a minimap. A detail panel shows device status, owner, tags, Relay metadata, and related relationships; keyboard users can select a Relay target and add it. Per-user node positions and viewport are saved in that browser's local storage with a versioned key. New nodes take the next free grid cell in their role column. Assignment drafts remain in memory and prompts appear before discard or navigation with unsaved changes.

Users may edit only source devices allowed by their server-side role and per-device `canEdit` decision. Edits are reviewed as one draft and submitted through `PUT /assignments/batch`. The backend validates every source and target before changing Policy, replaces only exact grants owned by Tailor, and preserves manually managed grants and other Policy fields. If a device IP changes, its old platform-managed exact grant is removed and the current IP pair is written. Duplicate exact policy entries, unsafe selector changes, missing devices, permission failures, and stale hashes reject the complete batch. `autogroup:self` is evaluated only as a destination, requires both Headscale user IDs, and excludes tagged destination devices. Nested groups use OR membership; an unresolved nested group does not prevent evaluation of later members, but the relationship retains an uncertainty warning. The existing one-device graph and assignment endpoints remain available for compatibility.

## Policy Write Safety

Policy writes are disabled by default (`PEER_RELAY_POLICY_WRITES=false`). Enable them only after confirming Headscale is configured for database-backed Policy and verifying Peer Relay support in the deployed version.

For each device-assignment or selector-rule write the backend:

1. Refreshes Headscale state and fetches the current full policy.
2. Compares the caller's base policy hash with the fresh policy; stale edits return a conflict.
3. Changes only the top-level `grants` array, retaining the other policy fields and comments outside that array.
4. Validates every changed source, target Relay, permission, IP selector, and existing grant shape before any Policy write. Selector writes are administrator-only, use one validated user or fully resolvable `group:` source selector and one registered Relay IPv4 destination, and require both the preview's policy hash and current source-scope hash. It refuses to create a missing `grants` array because that may change default access.
5. Merges the requested changes into one complete Policy document, saves a pending revision and audit record, writes once, and reads it back. The revision succeeds only when the semantic hash matches. Successful revision, per-device assignment ownership or selector-rule ownership, and completion audit records are committed together in SQLite.

Tailor serializes policy writes within one API process and performs a final policy read immediately before each write. Headscale's Policy REST endpoint does not provide compare-and-swap; a separate writer can still change Policy between that final read and the PUT. Deployments with multiple Tailor replicas must use one policy-writing replica or an external shared write lock.

Policy input supports HuJSON comments and trailing commas. Rewriting `grants` reformats that array and does not retain comments placed inside it; comments and fields outside it are retained. Unknown selectors, Headscale-unsupported IP sets or device postures, `autogroup:internet` without synchronized exit-node route data, and grants with `via`, extra capabilities, or other fields Tailor cannot preserve are displayed as read-only or uncertain. `autogroup:danger-all` is recognized as a broad all-IP source and carries a range warning; it is never accepted by the selector-rule writer. `tagOwners` is not a grant selector. The policy parser is intentionally not a general Headscale ACL editor.

Successful policy revisions keep the before/after snapshots and hashes; only the latest 30 successful revisions are retained. Admins can load one revision's snapshots on demand to preview added/removed Relay grants, affected selectors, and whether other Policy fields changed. Rollback selects only a successful revision and writes its `before` policy only if the live policy still matches the latest successful platform revision, protecting intervening external changes. Admin-only audit records expose before/after data and record pending, success, failure, or conflict outcomes for policy operations.

## API And Persistence

The API contract is `api/openapi/openapi.yaml`; routes are indexed in `docs/api/route-index.md`. Authenticated unsafe requests are limited to 60 per user per minute; OIDC entry points and public logout are limited to 20 per network peer per minute. The limiter is process-local, so multi-replica deployments need an edge or shared-store limit. SQLite stores OIDC login state, hashed Tailor sessions, Headscale snapshots, relay metadata, applications, per-device assignment ownership, selector-rule ownership, role/application settings, policy revisions, audit records, and sync state. Canvas layout is browser-local and adds no server table or API. GORM model migration runs on startup. Settings changes are audited.

## Frontend Module Boundaries

`frontend/src/app/router.tsx` mounts each page directly and owns browser paths, matching, navigation, and access/login redirects. `frontend/src/App.tsx` composes the sidebar, shared header, session status, and `Outlet`; `main.tsx` configures providers and connects query errors to the router handler. Each route page under `features/peer-relay/pages/` owns its content, queries, mutations, and dialogs. Shared cached query helpers live in `features/peer-relay/queries.ts`. Administrators can open the policy and settings routes, while applicants and reviewers can open the requests route. `frontend/src/components/app-sidebar.tsx` renders the responsive collapsible sidebar, `nav-main.tsx` renders page choices, and `nav-user.tsx` renders account controls. Each user-facing view remains a separate module under `frontend/src/features/peer-relay/pages/`. Feature dialogs and presentation helpers live under the feature's `components/` directory. Shared controls under `frontend/src/components/ui/` follow the configured shadcn aria-sera style and consume the global theme tokens in `frontend/src/index.css`.

## Known MVP Limits

- Single Headscale instance; no Tailscale control-plane provider.
- Authentication uses one configured OIDC issuer per deployment. OIDC group claims and IdP global logout are not supported.
- No relay health probe, automatic service enablement, route telemetry, or fastest-relay selection.
- The canvas v1 target is one Headscale instance with up to 500 devices. Dense relationships should be reviewed with representative deployment data. It does not provide a general Policy editor or shared server-side layouts.
- Existing single-device graph and assignment endpoints remain for compatibility. Peer Relay list APIs still return complete arrays; directory pagination is in the browser.
- Compatibility must be verified against the deployed Headscale build before enabling writes.
