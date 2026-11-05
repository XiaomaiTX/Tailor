---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docker-compose.example.yaml
  - backend/Dockerfile
  - frontend/Dockerfile
  - docs/architecture/auth-and-permissions.md
---

# Release Or Deploy Guide

## Scope

Compose starts the backend and frontend Nginx. The OIDC provider remains an external dependency; register Tailor as its own OIDC client before exposing the console.

## Preconditions

- A supported, version-pinned Headscale instance with an API key
- An OIDC issuer reachable by the backend, plus a Tailor client ID, secret, and exact callback URL registered at that issuer
- Network rules preventing untrusted clients from reaching backend directly
- HTTPS from browser to Tailor, and appropriate protected networking to backend
- For policy writes, verified Headscale database Policy mode and Peer Relay Grants support

## Steps

### 1. Configure secrets

```bash
cp .env.example .env
```

Set the deployment Headscale URL and API key, OIDC issuer URL, dedicated Tailor client ID, client secret, exact callback URL, database path, and public ports. Inject `OIDC_CLIENT_SECRET` through the deployment secret manager. Keep `PEER_RELAY_POLICY_WRITES=false` initially.

### 2. Build and start

```bash
docker compose --env-file .env -f docker-compose.example.yaml up --build -d
```

### 3. Register and configure OIDC

Register the configured callback URL (for example, `https://tailor.example.com/api/v1/auth/callback`) for Tailor's dedicated OIDC client. Point the public hostname to the frontend service and use HTTPS. The backend must be able to reach the issuer discovery, token, and JWKS endpoints. Keep the backend on the private Compose/deployment network.

### 4. Verify

- `GET /api/v1/health/live` returns `200`.
- An unauthenticated business API request returns one `401` Problem response.
- Complete SSO login with a verified email that uniquely matches a current Headscale user, then reach `/api/v1/relay/me`; verify the mapped role and Headscale username.
- Check dashboard sync freshness and device counts.
- Keep policy writes disabled until a review of a real deployment confirms read behavior and Headscale compatibility. Then enable `PEER_RELAY_POLICY_WRITES=true` only if database Policy mode is active.

## Operational Notes

- Back up the SQLite database securely; it contains audit history and full policy revision snapshots.
- Keep the deployment's Headscale version pinned and repeat compatibility checks before upgrading it.
- Failed syncs preserve the last successful snapshot; surface stale sync state to operators.
- Policy writes are serialized in one backend process. Do not run multiple backend replicas against the same Headscale policy without a distributed concurrency design.
- Rollback is guarded by the current policy hash and will refuse to overwrite policy edits made after the latest platform revision.

## Troubleshooting

- **401 from business routes**: the Tailor session is missing, expired, or maps to a user no longer present in the current Headscale snapshot; sign in again after verifying synchronization.
- **Login returns to `/login?error=sso`**: verify the callback URL, issuer/client configuration, verified email, and unique current Headscale email mapping.
- **Find the failed OIDC step**: inspect backend logs for `OIDC callback rejected` and its `stage`. The log contains no token or email values.
- **503 from business routes**: check SQLite availability, Headscale configuration/availability, and whether policy writes are enabled.
- **Sync stays stale**: check provider logs without printing credentials, REST API reachability, and the Headscale version's endpoint response shape.
- **Policy write fails**: confirm database Policy mode, Peer Relay support, current policy hash, and that external automation is not concurrently editing the same policy.

For local steps see `docs/guides/local-development.md`; security requirements are in `docs/standards/security-and-secrets.md`.
