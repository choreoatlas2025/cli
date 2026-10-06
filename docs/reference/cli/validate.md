# Validate Command Reference

## Overview

The `validate` command performs dynamic validation of FlowSpec against execution traces, ensuring that the actual service interactions match the declared choreography specifications.

## Usage

```bash
choreoatlas validate [options]
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--flow` | string | `.flowspec.yaml` | Path to FlowSpec file |
| `--trace` | string | *(required)* | Path to local native trace JSON (top-level `spans` array) |
| `--semantic` | bool | `true` | Enable semantic validation (CEL) |
| `--causality` | string | `temporal` | Causality check mode: `strict`, `temporal`, or `off` |
| `--causality-tolerance` | int | `50` | Nonnegative tolerance in milliseconds for completion dependencies and metadata timing constraints |
| `--baseline` | string | - | Path to baseline file for comparison |
| `--baseline-missing` | string | `fail` | Strategy when baseline file is missing: `fail` or `treat-as-absolute` |
| `--threshold-steps` | float | `0.9` | Absolute minimum step coverage in every mode (0.0-1.0) |
| `--threshold-conds` | float | `0.95` | Absolute minimum condition pass rate in every mode (0.0-1.0) |
| `--max-steps-degradation` | float | `0` | Maximum relative coverage degradation with a baseline (0.0-1.0) |
| `--max-conds-degradation` | float | `0` | Maximum relative condition pass rate degradation with a baseline (0.0-1.0) |
| `--skip-as-fail` | bool | `false` | Treat SKIP conditions as FAIL |
| `--max-input-bytes` | integer | `33554432` | Per-file byte limit; combined captured customer inputs are limited to twice this |
| `--max-spans` | integer | `10000` | Trace span limit |
| `--max-steps` | integer | `2000` | Contract call-step limit |
| `--max-cel-cost` | integer | `100000` | CEL expression and aggregate invocation cost limit, including output materialization |
| `--validation-timeout-ms` | integer | `10000` | Cooperative preparation/evaluation timeout; maximum one hour |
| `--report-format` | string | - | Report format: `json`, `junit`, or `html` |
| `--report-out` | string | - | Path for report output |

## Trace Input Format

CE reads local JSON in the repository's native trace format:

```json
{
  "spans": [
    {
      "name": "createOrder",
      "service": "orderService",
      "startNanos": 1,
      "endNanos": 2,
      "attributes": {}
    }
  ]
}
```

CE does not provide OTLP receivers, exporters, or direct import of OTLP JSON or
protobuf payloads. Convert other formats to native trace JSON outside CE.
OpenTelemetry-derived attributes in native spans can still be used for local
validation and causal checks.

One input represents one trace. If `attributes["otlp.trace_id"]` is supplied,
all spans must carry the same nonempty string ID. Mixed IDs, partially labelled
inputs, and malformed identity attributes fail validation, including in `off`
mode. Inputs with no trace IDs remain supported and reports mark their identity
as `file-only`; a file hash does not prove that all spans came from one request.

Native JSON integer attributes retain exact signed 64-bit values, or unsigned
64-bit values for positive integers above the signed maximum. Nested objects,
lists, CEL outputs, input references and text interpolation retain these values;
adjacent integer IDs above `2^53` do not become equal. Integers outside these
ranges are input errors (`2`). Decimal/exponent spellings retain `double`
arithmetic semantics. They use finite IEEE-754 binary64 approximation for
fractions, but integral values which would be rounded are rejected; use an
integer literal for exact integer identity. Overflow and nonzero underflow
also return `2`.

In `temporal` mode, each declared predecessor must complete before the next step
starts, within the configured tolerance. A step after a parallel group waits for
every member. `strict` instead verifies direct parent-child nesting and its time
bounds. Both `startNanos` and `endNanos` must be present for either timing mode.
An explicit integer `0` is a valid timestamp; an omitted field is not epoch
evidence. Null, noninteger and out-of-range timestamp fields are input errors
(`2`). Missing required fields, negative times and reversed intervals fail
validation (`3`), and cannot be recorded as a successful baseline.

`off` disables causality dependencies and permits serial matching without time
fields. It does not disable a Flow's explicit `parallel` overlap requirement:
such flows still require complete, valid interval evidence. DAG independent
nodes do not imply that their calls must overlap.

The validation engine receives an explicit `ValidationConfig` per call. It
does not change process-wide semantic, causality or tolerance settings, mutate
the supplied DAG while normalizing dependencies, or write to stdout. Causal
violations are returned with the step results; the CLI owns console output.
Concurrent calls must treat their supplied contract and trace as read-only.

DAG dependencies combine node `depends` and explicit `edges`. Supplying any
explicit edges never disables another node's `depends` constraints. Duplicate
identical edges are normalized before checks; missing references and cycles in
the combined graph are input errors. Runtime matching and output visibility use
the same combined graph as lint and conversion. CE supports unconditional edges
only. A nonempty `edge.condition` is an input error (`2`), including with
`lint --schema=false`, `--semantic=false`, or `--causality off`. An omitted or
empty condition means an unconditional edge.

## Exit Codes

The default contract checks are shared with `lint`, discovery and baseline
recording: the embedded FlowSpec and ServiceSpec schemas run before lint and
runtime matching. Empty titles or service names and duplicate `operationId`
values are input errors (`2`); definition order cannot choose which conditions
apply. The explicit `lint --schema=false` option skips schema checks only.

The validate command uses standardized exit codes for CI/CD integration:

| Code | Constant | Description |
|------|----------|-------------|
| `0` | `OK` | All validations and gates passed |
| `1` | `CLIError` | General CLI errors (invalid arguments, etc.) |
| `2` | `InputError` | File not found or parsing errors |
| `3` | `ValidationFailed` | Validation failures (spec vs trace mismatch) |
| `4` | `GateFailed` | Gate policy violations (thresholds not met) |

Any failed step or CEL compilation, type, or runtime error makes runtime validation fail with code
`3`, even if coverage thresholds pass. Code `4` applies only when runtime
validation passes and the threshold policy fails. Reports use the same final
decision: JSON and HTML expose `success`, `status`, and `exitCode`; JUnit includes
a failing policy test case when thresholds fail.

Declared conditions must return a boolean. Errors and unevaluated condition
results cannot produce a successful report; relaxing thresholds or leaving
`--skip-as-fail` disabled does not suppress these failures.

Baseline format `3` binds FlowSpec and every referenced ServiceSpec by SHA-256,
and records the validator version, commit and binary hash, semantic/causality
settings, source trace hash and available trace identity. Record a new baseline
after a contract, validator or validation setting changes, or when migrating from
format `1` or `2`. Comparison traces may differ from the source recording.
`treat-as-absolute` handles a missing file only; malformed or incompatible
baselines remain errors. Threshold values must be finite numbers between 0 and 1.

`baseline record` accepts `--semantic`, `--causality`, and
`--causality-tolerance` with the same defaults as `validate`. Use the same settings
for recording and consumption. Empty or whitespace-only titles cannot be recorded.

Reports bind contract files, trace data, the consumed baseline, validator binary,
and validation settings under `inputs` (JUnit: `result.inputs` property).

Within one invocation, Schema checks, parsing, runtime validation and input
hashes use the same captured bytes. Baseline loading, compatibility checks and
recording reuse that capture; reports never reopen these inputs to calculate
their identities. Each cleaned absolute path is read once. A later invocation
captures changed files afresh. The capture is not atomic across files and does
not deduplicate distinct symlink/hardlink paths by physical file identity;
prevent concurrent writes when a consistent multi-file revision is required.

## Qualified plans and resource limits

`lint`, `init` qualification and default discovery qualification compile the
contract's CEL conditions and outputs before accepting it. Syntax errors and
statically non-boolean conditions are input errors (`2`). Dynamic field access
and dynamic result types still require runtime evidence: an absent field or a
non-boolean runtime condition fails validation (`3`). Runtime validation and
baseline recording keep their existing code `3` for CEL preparation failures.
Explicit `--semantic=false` skips semantic qualification and evaluation for
runtime validation; it does not weaken structural checks.

A `ContractPlan` owns a copy of the contract and compiled programs. Duplicate
normalized expressions compile once per plan. Embedded callers can reuse a plan
concurrently; each invocation has its own variables, cost counter and context.
Changed contracts or settings require a new plan. There is no global compilation
cache. Reports include the plan hash and resource settings under `inputs`.

The resource flags also apply to `lint` and `baseline record`. Explicit flag
values must be positive. Defaults limit each input to 32 MiB, total captured
customer input to 64 MiB and input files to 128. The executing binary is exempt
from the customer byte budget. Span counts are checked during JSON decoding and
again for caller-supplied traces. Each expression is limited to 64 KiB and parser
recursion to 100; total expression text is limited to 1 MiB and declarations to
10,000. These limits also apply during generated-contract qualification.

CEL cost is an operation estimate, not elapsed time or a hard memory quota.
Each expression has the configured cost limit; aggregate actual CEL cost and
one unit per materialized output node are charged to the invocation. An
expression can finish before its aggregate cost is checked, but an overrun
always fails the result. Cancellation is cooperative: CEL comprehensions check
periodically, validation checks between stages/nodes, and preparation checks
between expressions. Parsing, individual compilation and other non-CEL work
are bounded by input/count limits but cannot be preempted at the deadline.

Baselines bind resource settings alongside semantic settings. After changing
limits, or upgrading from a baseline without these fields, record a new baseline.

## Cross-step values

With semantic validation enabled, a successful step evaluates its `output` CEL
expressions against `request`, `response`, `span` and `vars`. Subsequent steps can
read the exported values through `${name.field}` in `input` and `vars.name.field`
in conditions. Whole-value references retain their number, boolean, object or
list type. References embedded in text accept scalar values only. Flat input
maps appear as `request.body`; explicit `body`, `path`, `query` or `headers`
maps retain that request structure.

Flow steps publish outputs after their stage. Parallel siblings cannot read
each other's outputs. DAG nodes see only their ancestors' successful outputs;
sequential descendants may replace an earlier value, but independent predecessors
exporting the same variable are ambiguous and fail. A failed step publishes no
outputs. Failed output expressions fail the step with code `3`.

Initial external variables have no injection option. Reading an unresolved input
reference produces a CEL error; an unused input does not prevent an independent
response assertion from being evaluated. `--semantic=false` disables both
conditions and output evaluation. Values persist only for the current invocation.

## Examples

### Basic Validation

```bash
choreoatlas validate \
  --flow order-flow.flowspec.yaml \
  --trace traces/order-123.json
```

### With Gate Thresholds

```bash
choreoatlas validate \
  --flow order-flow.flowspec.yaml \
  --trace traces/order-123.json \
  --threshold-steps 0.95 \
  --threshold-conds 0.98
```

### Generate JUnit Report for CI

```bash
choreoatlas validate \
  --flow order-flow.flowspec.yaml \
  --trace traces/order-123.json \
  --report-format junit \
  --report-out test-results.xml
```

### Compare with Baseline

```bash
choreoatlas validate \
  --flow order-flow.flowspec.yaml \
  --trace traces/current.json \
  --baseline baseline.json
```

## Validation Process

1. **Lint Check**: Static validation of FlowSpec consistency
2. **Trace Loading**: Parse and validate trace.json format
3. **Dynamic Validation**: Match trace spans against FlowSpec steps
4. **Semantic Validation**: Evaluate CEL conditions (if enabled)
5. **Causality Check**: Verify temporal ordering (if enabled)
6. **Gate Evaluation**: Check coverage and pass rate thresholds
7. **Report Generation**: Output results in requested format

## See Also

- [README Exit Codes](../../../README.md#exit-codes)
- [FlowSpec Schema](../../flowspec/schema.md)
- [CI Integration Guide](../../ci/github-actions.md)
