---
title: "CE: Telemetry→Contract mapping (keep FlowSpec.input clean)"
labels: ["type:enhancement", "area:cli", "scope:ce", "priority:P0"]
---

# Summary
Move telemetry attributes (`otel.*`, `http.*`, `span.*`) out of FlowSpec `input` and synthesize them into ServiceSpec `preconditions`/`postconditions`. Limit FlowSpec inputs to actual call arguments (path/query/header/body).

# Problem
- `discover` injects raw span attributes as request `input.body` and outputs variables like `<service>Response`.
- This violates FlowSpec semantics and produces invalid/fragile contracts.

# Proposal
- Default mapping policy in CLI:
  - Telemetry attributes → ServiceSpec assertions:
    - Preconditions: method, route, required headers, basic request invariants.
    - Postconditions: status code, basic response invariants.
  - FlowSpec `input`: only accepted fields are `path`, `query`, `headers`, `body` (actual call params).
  - Drop null/empty telemetry values; mask sensitive values by simple rules (e.g., `mask_query` for `http.url`).
- Provide `--mapping` to load a YAML with include/rename/drop rules (optional for CE but supported).

# CE Scope
- File‑in/file‑out CLI only; no Kafka/Registry/UI.

# Acceptance Criteria
- Generated FlowSpec has no keys prefixed by `otel.`, `http.`, or `span.` under `input`.
- Corresponding ServiceSpec includes assertions for `http.method`, `http.route`, and `http.response.status_code` when available in trace.
- JSON Schema + lint pass out‑of‑box on generated artifacts.

# Implementation Notes
- Extend `spec.GenerateServiceSpecs` to derive pre/postconditions from attributes (partially exists) and enrich rules:
  - Recognize `http.method`, `http.route`, `http.response.status_code`, `user_agent`, `authorization`.
  - Ensure conditions use the engine’s field names (e.g., `response.status`).
- Update `internal/cli/discover.go` to stop dumping attributes into FlowSpec `input` by default.
- Introduce minimal redaction helpers for URLs/headers.

# Touch Points
- internal/cli/discover.go
- internal/spec/service.go (condition synthesis rules)
- internal/validate/static.go (input key whitelist if needed)

# Test Plan
- Golden sample: Jaeger/OTLP trace for GET /health → FlowSpec with clean input and ServiceSpec with assertions; both pass schema and lint.
- Negative sample: attributes with null/empty are ignored; no schema warnings.

# References
- discover generation today: choreoatlas_cli/internal/cli/discover.go:1
- service synthesis today: choreoatlas_cli/internal/spec/service.go:1

