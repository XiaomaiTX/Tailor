---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/internal/
  - frontend/src/
  - api/openapi/openapi.yaml
---

# Module Map

## Scope

Maps Tailor's modules and their dependency boundaries. Layering rules are in `docs/standards/dependency-layering.md`.

## Backend

| Module          | Entry                     | Responsibility                                                                 | Dependencies                                          |
| --------------- | ------------------------- | ------------------------------------------------------------------------------ | ----------------------------------------------------- |
| `cmd/api`       | `backend/cmd/api/main.go` | Configuration, migration, dependency assembly, background sync, HTTP lifecycle | config, model, headscale, repository, service, router |
| `config`        | `config.Load`             | Environment configuration and validation                                       | environment                                           |
| `router`        | `router.New`              | Gin middleware and route registration                                          | handler, middleware, generated API, service           |
| `middleware`    | `AuthSession`             | Session enforcement, request ID, security headers, CORS, logging, recovery     | Gin, config, Problem                                  |
| `handler`       | `RegisterAuthRoutes`      | OIDC login, callback, and local session logout HTTP mapping                    | service, middleware, Problem                          |
| `handler`       | `RegisterPeerRelayRoutes` | Strict JSON parsing and HTTP/service mapping                                   | service, middleware, Problem                          |
| `service`       | `OIDCService`             | OIDC validation, verified-email mapping, and opaque session lifecycle          | OIDC provider, OAuth2, repository, model               |
| `service`       | `PeerRelayService`        | Roles, ownership, sync, lifecycle, graph, policy commit, revision, audit       | repository, Headscale interface, policy, model        |
| `headscale`     | `NewClient`               | REST provider for Headscale user, node, and Policy APIs                        | net/http, config values                               |
| `policy`        | `Parse`                   | HuJSON decoding, grant and selector classification, safe grant mutation        | JSON and standard library                             |
| `repository`    | `NewAuthRepository`       | SQLite login state, session persistence, expiry, and revocation                 | GORM, model                                           |
| `repository`    | `NewPeerRelayRepository`  | SQLite snapshots and Peer Relay reads/writes                                   | GORM, model                                           |
| `model`         | Auth and Peer Relay structs | SQLite persistence mapping                                                   | GORM tags                                             |
| `problem`       | `problem.Write`           | RFC 7807 response writing                                                      | Gin, net/http                                         |
| `api/generated` | OpenAPI Go output         | Strict generated health/example handler                                        | oapi-codegen                                          |

## Frontend

| Module                  | Entry                                                     | Responsibility                                                                            | Dependencies                                                    |
| ----------------------- | --------------------------------------------------------- | ----------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `App`                   | `frontend/src/App.tsx`                                    | SPA entry that delegates to the Peer Relay console                                        | React Router, feature entry                                     |
| `peer-relay console`    | `frontend/src/features/peer-relay/peer-relay-console.tsx` | Query and mutation orchestration, role-aware navigation, shared page shell                | React Query, i18n, feature pages, API wrapper, UI components    |
| `peer-relay pages`      | `frontend/src/features/peer-relay/pages/`                 | Overview, graph editor, relay directory, applications, policy history, and settings views | Generated models, feature view components, shared UI primitives |
| `peer-relay components` | `frontend/src/features/peer-relay/components/`            | Application and review dialogs plus Peer Relay display primitives                         | React Aria, shadcn aria-sera UI primitives, feature types       |
| `api/peer-relay`        | `frontend/src/api/peer-relay.ts`                          | Typed domain-oriented calls around generated client                                       | generated client/models                                         |
| `api/generated`         | `frontend/src/api/generated/`                             | Orval client, React Query hooks, schemas                                                  | OpenAPI, shared fetch                                           |
| `i18n`                  | `frontend/src/i18n/`                                      | English and Simplified Chinese messages                                                   | React context                                                   |
| `components/ui`         | `frontend/src/components/ui/`                             | Shared aria-sera controls and surfaces themed with global CSS tokens                      | React Aria Components, Tailwind, global theme                   |
| `components`            | `frontend/src/components/`                                | Shared preference controls and theme provider                                             | UI primitives, Zustand                                          |

## Contract And Runtime

| Area             | Source                          | Responsibility                                                                          |
| ---------------- | ------------------------------- | --------------------------------------------------------------------------------------- |
| API contract     | `api/openapi/openapi.yaml`      | Request, response, and trust declarations                                               |
| Go generation    | `api/openapi/oapi-codegen.yaml` | Generate only health and Example handler interfaces                                     |
| React generation | `frontend/orval.config.ts`      | Generate client for the full contract                                                   |
| Compose          | `docker-compose.example.yaml`   | Backend/frontend runtime wiring and external OIDC configuration                |

## Boundaries

- Backend flow: `handler -> service -> repository -> model`; Headscale and policy parsing are service adapters.
- Frontend flow: `App -> feature console -> page/component -> API wrapper -> generated client -> HTTP transport`.
- Only generated folders are written by generators.
- Keep the backend on a private network and use HTTPS for deployed browser traffic and OIDC callbacks.
