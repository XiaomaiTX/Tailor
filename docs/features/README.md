---
status: active
doc_type: index
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/doc-writing-standard.md
---

# Feature Index

## Purpose

`docs/features/current/` describes behavior implemented in Tailor. Architecture and API details belong in their corresponding documentation directories.

## Current Features

- [Peer Relay management](current/peer-relay-manager.md): Headscale sync, relay registration and applications, approval, authorization graph, policy-safe writes, history, rollback, and audit.
- [Example resource flow](current/example-feature.md): retained starter CRUD API used to illustrate the generated handler path; it is not part of the main console workflow.

## Rules

- Current behavior belongs in `current/`.
- Superseded feature documents belong in `archived/`.
- Proposals that are not implemented belong in `docs/specs/draft/`.
