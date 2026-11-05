---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docs/architecture/runtime-and-startup.md
  - docs/standards/testing-and-verification.md
  - Makefile
---

# Local Development Guide

## Preconditions

- Go 1.27.3 and a C compiler (SQLite driver uses CGO)
- Node.js 24.x and pnpm 12.17.0
- A reachable Headscale instance and API key for real device/policy data
- A reachable OIDC issuer and a dedicated development client with a loopback redirect URL

## Setup

```bash
cp .env.example .env
```

Set `DATABASE_PATH`, `CORS_ORIGINS`, `HEADSCALE_URL`, `HEADSCALE_API_KEY`, and all four OIDC values. Register `http://localhost:5173/api/v1/auth/callback` as a redirect URL at the development issuer, so Vite can proxy the callback to the backend and return the browser to the SPA. Use its dedicated client ID and secret. For safe local inspection without policy edits, leave `PEER_RELAY_POLICY_WRITES=false`.

Install frontend dependencies:

```bash
cd frontend
pnpm install --frozen-lockfile
```

## Start

In separate terminals:

```bash
make backend-dev
```

```bash
cd frontend
pnpm dev
```

Vite's default `/api` proxy forwards requests to the backend. The backend redirects login to the configured issuer and keeps the resulting session in SQLite; no identity headers or browser-held tokens are used.

## Expected Outcome

- `GET http://localhost:8080/api/v1/health/live` responds without auth.
- `GET http://localhost:8080/api/v1/relay/me` responds after OIDC login when the verified email uniquely matches a synchronized Headscale user.
- The console displays data from the most recent successful Headscale sync.

## Useful Commands

```bash
make openapi-lint
make backend-build
make frontend-build
```

Testing commands and policy live in `docs/standards/testing-and-verification.md`.

## Troubleshooting

- **Startup reports OIDC configuration/discovery failure**: check the issuer URL and network reachability, client values, and redirect URL.
- **Business API returns 503**: OIDC session storage is unavailable, Headscale is unconfigured/unavailable, or Policy writes were requested while disabled.
- **Login returns to `/login?error=sso`**: check callback URL registration, issuer/client configuration, verified email, and a unique matching Headscale user email.
- **Find the failed OIDC step**: inspect backend logs for `OIDC callback rejected` and its `stage`. `verified_email` means the provider did not return a verified email for the same subject; `headscale_email_no_match` or `headscale_email_ambiguous` means the current successful Headscale snapshot has no unique email match; `state`/`state_consume` points to a missing, expired, or replayed callback; `token_exchange` or `id_token_validation` points to the IdP client, redirect, PKCE, issuer, audience, signature, or token-time checks. The log omits tokens and email values.
- **Business API returns 401**: the Tailor session is absent or expired; use the SSO login button again.
- **Business API returns 403**: the identity is not a current Headscale user or lacks the route's role/ownership.
- **Sync error in dashboard**: check Headscale URL, API key, network reachability, and deployed REST response format.
- **Policy save conflicts**: reload the graph and retry against the latest policy; do not overwrite external policy edits.
