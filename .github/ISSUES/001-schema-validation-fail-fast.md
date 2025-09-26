---
title: "CE: Add Schema Validation + Fail‑Fast Gate in CLI pipeline"
labels: ["type:enhancement", "area:cli", "scope:ce", "priority:P0"]
---

# Summary
CLI emits non‑compliant FlowSpec/ServiceSpec that “look successful”. We must enforce JSON Schema + structural lint in the main generation path and fail fast with actionable errors.

# Problem
- Generated FlowSpec misses `version/info`, uses invalid `call` (e.g., "market-data-service.GET /health"), and injects telemetry (`http.*`, `otel.*`) into `input`.
- `discover` currently writes files without schema guardrails; validation is only available via `lint` and not enforced as a gate.

# Proposal
- Bundle and use embedded schemas for FlowSpec/ServiceSpec to validate outputs of `discover` before writing.
- On violation: print clear diagnostics and exit non‑zero; support `--no-validate` to bypass (default off).
- Add a dedicated `flowspec validate` alias (or enhance `lint`) for directories and single files.

# CE Scope
- Pure offline, no network. Use embedded schemas in `internal/schemas`.
- Apply to file‑in/file‑out path; no registry/Kafka.

# Acceptance Criteria
- `ca discover --trace ...` aborts on invalid output unless `--no-validate`.
- Errors include: missing `info.title`, both `graph` and `flow` set, invalid `call` format, illegal keys under `input`, null values (e.g., `http.scheme: null`).
- `ca lint --schema` validates FlowSpec and all referenced ServiceSpec using embedded schemas and exits non‑zero on ERRORs.

# Implementation Notes
- Reuse `spec.ValidateYAMLWithSchemaFS` and schemas in `internal/schemas`.
- Integrate post‑generation validation in `internal/cli/discover.go` after writing to temp buffer; only persist when valid.
- Keep structural lint (`internal/validate/static.go`) in the same gate.

# Touch Points
- internal/cli/discover.go
- internal/cli/lint.go
- internal/spec/schema_fs.go, internal/schemas/*

# Test Plan
- Unit tests with sample invalid contracts to ensure non‑zero exit.
- Golden tests for a valid minimal FlowSpec + ServiceSpec.

# References
- flowspec schema: schemas/flowspec.schema.json
- servicespec schema: schemas/servicespec.schema.json
- Validator: choreoatlas_cli/internal/spec/schema_fs.go:1

