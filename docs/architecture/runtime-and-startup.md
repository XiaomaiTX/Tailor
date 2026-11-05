---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/cmd/api/main.go
  - backend/internal/config/config.go
  - backend/internal/middleware/auth_session.go
  - backend/internal/service/oidc.go
  - docker-compose.example.yaml
---

# Runtime And Startup

## Startup Flow

The backend (`backend/cmd/api/main.go`) does the following:

1. Loads and validates environment configuration.
2. Opens SQLite and auto-migrates Example, Peer Relay, OIDC login-state, and Tailor session models.
3. Discovers the configured OIDC provider, constructs the Headscale REST client, repositories, OIDC session service, Peer Relay service, existing Example service, and router.
4. Starts a background sync loop immediately and then at the configured interval (default 60 seconds).
5. Serves HTTP on `:$API_PORT` (default 8080), with graceful shutdown on SIGINT/SIGTERM.

The frontend is a React SPA. Vite proxies `/api` to the local backend; Compose's Nginx serves static files and proxies `/api/` to the backend container.

## External Runtime Dependencies

- A single Headscale instance and API key are needed for snapshots and graph data.
- A reachable OIDC issuer is required for browser login. Tailor validates OIDC ID tokens and maps verified email to a synchronized Headscale user; Headscale API access continues to use a backend-only API key.
- Headscale must use database-backed Policy mode before enabling policy writes.
- SQLite file path comes from `DATABASE_PATH`.

## Configuration

Required variables include `DATABASE_PATH`, `CORS_ORIGINS`, and the four OIDC settings. OIDC discovery is validated during startup; Headscale URL/key must be provided together or both omitted. Tailor's OIDC session lifetime is 12 hours.

Peer Relay variables:

| Variable | Default | Meaning |
| --- | --- | --- |
| `HEADSCALE_URL` / `HEADSCALE_API_KEY` | unset | Headscale base URL and backend-only API key; HTTPS is required except for local loopback development |
| `HEADSCALE_SYNC_INTERVAL_SECONDS` | `60` | Initial/fallback background sync interval (10–3600); admins can change the persisted interval in the system settings page |
| `OIDC_ISSUER_URL` | required | OIDC discovery issuer URL; HTTPS required except local loopback development |
| `OIDC_CLIENT_ID` | required | Tailor's dedicated OIDC client ID |
| `OIDC_CLIENT_SECRET` | required | Tailor's OIDC client secret; deployment secret manager only |
| `OIDC_REDIRECT_URL` | required | Exact callback URL registered at the IdP (`/api/v1/auth/callback`) |
| `PEER_RELAY_BOOTSTRAP_ADMINS` | empty | Comma-separated Headscale usernames with admin role |
| `PEER_RELAY_ADMIN_GROUPS` | `group:relay-admin` | Initial/fallback comma-separated admin groups; persisted settings are managed in the admin UI |
| `PEER_RELAY_REVIEWER_GROUPS` | `group:relay-reviewer` | Initial/fallback reviewer groups; persisted settings are managed in the admin UI |
| `PEER_RELAY_APPLICANT_MODE` | `all` | Initial/fallback application mode (`all`, `groups`, or `disabled`); persisted settings are managed in the admin UI |
| `PEER_RELAY_APPLICANT_GROUPS` | `group:relay-applicant` | Initial/fallback groups allowed to apply; persisted settings are managed in the admin UI |
| `PEER_RELAY_SELF_SERVICE_ENABLED` | `true` | Initial/fallback switch for user-owned device authorization; persisted setting is managed in the admin UI |
| `PEER_RELAY_POLICY_WRITES` | `false` | Explicitly enables Headscale policy writes |

`.env.example` documents the complete Compose configuration. `HEADSCALE_API_KEY` and `OIDC_CLIENT_SECRET` must be stored in a secret manager for deployment, not browser-visible configuration.

After a successful Policy sync, admins can edit role groups, application mode, self-service authorization, reviewer delegation, sync interval, and reapplication cooldown in the system settings page. The settings row is persisted in SQLite and takes precedence over the environment defaults. Headscale credentials and Policy write enablement remain environment-managed and read-only in the UI.

## Health Probes

- `GET /api/v1/health/live`: process liveness, public.
- `GET /api/v1/health/ready`: SQLite readiness, public.
- Headscale sync errors are reported in the dashboard sync state. They do not make SQLite readiness fail.

## Runtime Modes

- **Local development:** run backend and frontend separately; configure an OIDC test or development client with a loopback redirect URL and the Headscale API credentials.
- **Compose:** backend and frontend Nginx only. The OIDC provider is external; register Tailor as a dedicated client and use HTTPS in deployment.
- **Shutdown:** cancel the sync loop and allow the HTTP server 10 seconds to drain requests.

See `docs/guides/local-development.md` and `docs/guides/release-or-deploy.md` for operator steps.
