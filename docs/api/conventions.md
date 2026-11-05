---
status: active
doc_type: api
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - api/openapi/openapi.yaml
  - backend/internal/router/router.go
  - backend/internal/handler/peer_relay.go
  - docs/architecture/auth-and-permissions.md
---

# API Conventions

## Scope

Describes the `/api/v1` HTTP/JSON interface. `route-index.md` lists operations; `api/openapi/openapi.yaml` is the contract source.

## Boundary And Trust

- All routes use JSON except empty `204` responses; errors use `application/problem+json`.
- `/health/live`, `/health/ready`, and the OIDC login/callback endpoints are public. Business routes require the server-side session cookie established by OIDC login.
- OIDC login starts at `/auth/login`, returns through `/auth/callback`, and ends locally at `/auth/logout`. The backend verifies the provider token and maps a verified email to a unique current Headscale user.
- Unsafe browser requests require an allowed configured Origin. Session cookies are HTTP-only; Headscale credentials are never part of the HTTP contract or frontend configuration.
- Headscale API keys are never part of the HTTP contract or frontend configuration.

## Request And Response Notes

- OpenAPI declares the contract, including the Tailor session cookie security scheme and response schemas.
- Health and retained Example routes use the generated strict Go handler. Peer Relay routes use a dedicated Gin adapter with a 1 MiB body limit and unknown-field rejection; both call service logic and map errors to shared Problems.
- Service errors map to validation (`400`), unauthorized (`401`), forbidden (`403`), not found (`404`), conflict (`409`), unavailable (`503`), or internal (`500`) Problems.
- Example lists use `{ page, pageSize, list, total }`; current Peer Relay collections do not use pagination.
- Time fields use ISO 8601 `date-time`.

## Generation

- React API client and models are generated from OpenAPI with Orval.
- Go code generation is configured for `health` and `examples` tags. Peer Relay operations are explicitly excluded from generated Go interfaces and bound by `internal/handler/peer_relay.go` to keep the REST adapter separate from the existing generated starter handler.
- Keep the generated frontend contract synchronized with the YAML; do not hand-edit generated files.

## Versioning

- Only `/api/v1` currently exists.
- Breaking changes require a new version while consumers migrate; additive changes stay in v1.

## Synchronization Requirements

1. Update `api/openapi/openapi.yaml`.
2. Update provider, service, handler, and React call wrapper.
3. Regenerate React client/model types and Go code only for configured generated tags.
4. Update `docs/api/route-index.md` and the relevant feature/architecture documents.
5. Run lint and generation checks where tools are available and record any environment limitation.
