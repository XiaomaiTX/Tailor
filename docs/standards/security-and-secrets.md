---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/architecture/auth-and-permissions.md
  - docs/architecture/runtime-and-startup.md
---

# Security And Secrets Standard

## Current Posture

- Business routes require a valid Tailor server-side session. Tailor is an OIDC client and maps the provider's verified email to exactly one current Headscale user.
- Health probes alone are public. CORS and security response headers are defense-in-depth, not access control.
- Headscale API keys and the OIDC client secret are backend-only secrets. OIDC tokens are never returned to the browser.
- Policy writes default to disabled and require database-backed Headscale Policy mode.

See `docs/architecture/auth-and-permissions.md` for the enforcement model.

## Core Rules

- Never commit real API keys, tokens, passwords, certificates, or production policy snapshots.
- Keep `.env`, `data/`, and local certificates out of version control.
- `.env.example` contains placeholders only; replace every placeholder before deployment.
- Register a dedicated OIDC client for Tailor, even when Headscale uses the same issuer. Store `OIDC_CLIENT_SECRET` in a deployment secret manager.
- Configure the exact Tailor callback URL at the issuer and use HTTPS outside local loopback development.
- Give Headscale API keys the narrowest available scope; do not expose them to the browser, build arguments, logs, or errors.
- Use HTTPS for `HEADSCALE_URL` except when targeting a local loopback address. The API key is sent as a bearer token and must not traverse an unencrypted network.
- The API applies process-local limits to unsafe requests and OIDC entry points. Multi-replica deployments also need an edge or shared-store rate limit.

## Sensitive Variables

- `HEADSCALE_API_KEY`
- `OIDC_CLIENT_SECRET`
- `PEER_RELAY_BOOTSTRAP_ADMINS` (identity configuration)
- `DATABASE_PATH` content, including retained policy snapshots and audit metadata
- `CORS_ORIGINS` (deployment topology)

Compose should receive real values from deployment secret injection or a protected `.env`; Compose files contain variable references only.

## Sensitive Data Handling

- Keep persisted Headscale and policy data limited to what is needed for the feature.
- Do not log API keys, OIDC client secrets or tokens, full Headscale responses, or full Policy documents.
- Error responses must not reveal internal hosts, credentials, or stack traces.
- Use HTTPS between the browser and Tailor, and secure the backend network and OIDC provider connection in deployment.
- Restrict SQLite file and backup access; policy revision rows contain full policy snapshots.

## Required Review When Changing Auth Or External Access

1. Confirm the server enforces identity and ownership; UI-only checks do not count.
2. Verify OIDC state, nonce, issuer, audience, signature, expiry, verified-email mapping, and session revocation.
3. Update `docs/architecture/auth-and-permissions.md`, API trust docs, and deployment guidance.
4. Check `.gitignore`, `.env.example`, Compose injection, logs, and error handling.
5. Add behavior-level tests for changed permission decisions when test changes are permitted.
