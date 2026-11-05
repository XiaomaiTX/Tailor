---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/internal/middleware/auth_session.go
  - backend/internal/service/oidc.go
  - backend/internal/service/peer_relay.go
  - backend/internal/config/config.go
  - api/openapi/openapi.yaml
---

# Auth And Permissions

## Scope

Describes the current identity trust boundary and server-enforced Peer Relay permissions.

## Identity Integration

Tailor is an OIDC client. Each deployment configures one external issuer and registers a dedicated Tailor client with its own client ID, client secret, and redirect URI. The application uses the authorization-code flow with PKCE, state, and nonce. The callback verifies the ID token's signature, issuer, audience, expiration, and nonce, then requires a verified email claim.

The verified email is normalized and matched case-insensitively to exactly one email in the current synchronized Headscale users. Missing, unverified, unmatched, or ambiguous emails are denied. The resulting Headscale user ID is stored in the server-side SQLite session; each business request resolves that ID to the current Headscale username before entering service authorization.

`GET /api/v1/auth/login` starts OIDC login, `GET /api/v1/auth/callback` validates the provider response and creates a session, and `POST /api/v1/auth/logout` revokes the local Tailor session. The session cookie is opaque, `HttpOnly`, `SameSite=Lax`, and `Secure` when the configured redirect uses HTTPS. The browser never receives the ID token, access token, or Headscale API key. Logout does not end the IdP session. Origin checks protect unsafe cookie-authenticated requests, and process-local rate limits bound OIDC entry points and authenticated mutations.

OIDC configuration is environment-managed: `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, and `OIDC_REDIRECT_URL` are required. Register Tailor as a separate client at the same IdP Headscale may use; Tailor does not reuse Headscale's OAuth client or token. Business requests require a valid Tailor session; only `/api/v1/health/live` and `/api/v1/health/ready` are public. The authentication middleware aborts the Gin chain after writing an error response.

## Roles And Enforcement

The service derives roles from the current Headscale policy `groups` map plus server configuration:

| Role | Source | Permissions |
| --- | --- | --- |
| Admin | configured admin groups (default `group:relay-admin`) or `PEER_RELAY_BOOTSTRAP_ADMINS` | Manage settings, relays, and all eligible device graphs; review; sync; inspect all applications, revisions, and audit |
| Reviewer | configured reviewer groups (default `group:relay-reviewer`); admins are also reviewers | Inspect applications and approve/reject requests; admins can optionally delegate Relay management and all-device graph editing |
| Applicant | `PEER_RELAY_APPLICANT_MODE=all` (default), membership in configured applicant groups when `groups`, or disabled | Apply for an owned untagged device, inspect own requests, and cancel a pending request |
| Regular user | current Headscale user without elevated groups | Inspect own devices; edit an owned untagged device graph only when self-service authorization is enabled and Policy writes are enabled |

User and device ownership are resolved against synchronized Headscale IDs and node snapshots. Regular users cannot use a client-supplied owner field to access another user's devices. Tagged devices are admin-managed. Admin and reviewer checks are performed in the service, not inferred from UI visibility. Delegated reviewers who can manage relays but cannot edit all devices may register only their own devices. The self-service switch restricts ordinary users; admins and explicitly delegated reviewers retain their respective management permissions.

## Policy And Relay Boundaries

- Relay registration, application review, graph edits, sync, revision history, rollback, and audit routes require a valid OIDC-backed Tailor session.
- Platform roles continue to come from Headscale Policy Groups and server configuration; OIDC `groups` claims are not used.
- Policy writes are additionally disabled unless `PEER_RELAY_POLICY_WRITES=true`.
- The network authorization canvas scopes source nodes in the service: admins and reviewers with all-device graph permission can inspect all devices; regular users can inspect their own source devices. Registered Relay targets are visible to authenticated users. Batch writes re-check each source device, tag state, Relay registration/selectability, IP availability, and the current Policy hash.
- Headscale API credentials are backend-only. The React app never receives the API key.
- Non-platform policy grants, unknown selectors, inherited relationships, and duplicate ambiguous exact grants are not silently rewritten.
- Rollback is refused if the current live policy differs from the latest successful platform revision.
- CORS and browser security headers are not authorization controls.

## Invariants

- All business API routes remain inaccessible without a valid server-side session mapped to a current Headscale user.
- Authorization is checked in middleware and service layers; frontend permission checks are UX only.
- Any identity or permission change must update this document, API trust contracts, feature behavior documentation, and an appropriate behavior-level test.
- See `docs/standards/security-and-secrets.md` for credential handling and `docs/features/current/peer-relay-manager.md` for feature behavior.
