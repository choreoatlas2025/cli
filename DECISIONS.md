# Repository Decisions

## 2026-10-04: Independent Community Edition

This repository is the standalone Community Edition (CE), licensed under
Apache-2.0. The default build and release pipeline produce CE executables.
Version output and HTML reports identify the executable as CE.

The scope includes:

- ServiceSpec and FlowSpec parsing, discovery, and validation.
- CEL conditions, temporal and causal checks, and DAG validation.
- HTML, JSON, and JUnit reports and coverage summaries.
- Basic baseline recording, threshold gates, and local baseline comparison.
- Local project initialization and CI examples.

Keep the code, build configuration, documentation, and reports scoped to these
local capabilities. Expose only implemented CLI commands. Core builds must not
require an edition selection, license activation, remote configuration, or
outbound network connection. Compatibility no-op telemetry interfaces must
remain inert and available in the default build.

OpenTelemetry span attributes are local validation inputs. Their parsing and
causal checks belong to the core; they are not product usage collection.
Baseline coverage comparisons also belong to the core and do not constitute a
historical analysis service.

Scope changes must update this decision and document their compatibility impact.
Use the README and privacy policy to describe the current executable's behavior.
