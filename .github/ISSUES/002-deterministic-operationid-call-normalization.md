---
title: "CE: Deterministic operationId + call normalization (HTTP/RPC)"
labels: ["type:enhancement", "area:cli", "scope:ce", "priority:P0"]
---

# Summary
Normalize HTTP/RPC names into stable, spec‑safe `operationId` and `call = serviceAlias.operationId`. Today `discover` uses raw span names (e.g., "GET /health").

# Problem
- `internal/cli/discover.go` writes `call: "<service>.<span.Name>"`, which may contain spaces and slashes.
- `normalizeOperationName` in `internal/spec/service.go` collapses non‑alphanumerics and produces awkward ids (e.g., "gEThealth").

# Proposal
- HTTP mapping: `{METHOD} /path/tokens/{id}` → `getPathTokensById`. Rules:
  - Method lowercased prefix: get|post|put|patch|delete|options|head.
  - Path tokens PascalCase; `{param}` → `ByParam` suffix chain.
  - Dedup consecutive slashes, ignore trailing slash.
- RPC mapping: `rpc.service + rpc.method` → `serviceMethod` (camelCase).
- Collision handling: append stable numeric suffix `_<n>` when necessary.
- `call` format: `serviceAlias.operationId` (strict).

# CE Scope
- Default behavior in `discover`; expose optional `--http-naming-style` for future customization (not required now).

# Acceptance Criteria
- “GET /health” → `getHealth`; “GET /users/{id}” → `getUsersById`; “POST /orders/{order_id}/items/{item_id}” → `postOrdersByOrderIdItemsByItemId`.
- No spaces or slashes in `operationId`/`call`.
- Stable across runs with same input.
- Unit tests covering edge cases (empty path, multiple params, hyphens/underscores).

# Implementation Notes
- Add helper in `internal/spec` (e.g., `httpop.Normalize(method, route) string`).
- Update `internal/cli/discover.go` to derive method+route from span attributes when available; fallback to span name parsing.
- Keep existing `splitCall` rules in `internal/validate/static.go`.

# Touch Points
- internal/cli/discover.go: use normalized operationId for both ServiceSpec generation and FlowSpec calls.
- internal/spec/service.go: replace/extend `normalizeOperationName` with HTTP/RPC aware version.

# Test Plan
- New unit tests for normalization helper covering typical/edge inputs.
- Golden tests for generated FlowSpec to assert `call` strings.

# References
- discover generation: choreoatlas_cli/internal/cli/discover.go:1
- current normalizer: choreoatlas_cli/internal/spec/service.go:240

