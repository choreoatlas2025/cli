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
| `--causality-tolerance` | int | `50` | Nonnegative tolerance in milliseconds for metadata timing constraints |
| `--baseline` | string | - | Path to baseline file for comparison |
| `--baseline-missing` | string | `fail` | Strategy when baseline file is missing: `fail` or `treat-as-absolute` |
| `--threshold-steps` | float | `0.9` | Absolute minimum step coverage in every mode (0.0-1.0) |
| `--threshold-conds` | float | `0.95` | Absolute minimum condition pass rate in every mode (0.0-1.0) |
| `--max-steps-degradation` | float | `0` | Maximum relative coverage degradation with a baseline (0.0-1.0) |
| `--max-conds-degradation` | float | `0` | Maximum relative condition pass rate degradation with a baseline (0.0-1.0) |
| `--skip-as-fail` | bool | `false` | Treat SKIP conditions as FAIL |
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

## Exit Codes

The validate command uses standardized exit codes for CI/CD integration:

| Code | Constant | Description |
|------|----------|-------------|
| `0` | `OK` | All validations and gates passed |
| `1` | `CLIError` | General CLI errors (invalid arguments, etc.) |
| `2` | `InputError` | File not found or parsing errors |
| `3` | `ValidationFailed` | Validation failures (spec vs trace mismatch) |
| `4` | `GateFailed` | Gate policy violations (thresholds not met) |

Any failed step or evaluated CEL condition makes runtime validation fail with code
`3`, even if coverage thresholds pass. Code `4` applies only when runtime
validation passes and the threshold policy fails. Reports use the same final
decision: JSON and HTML expose `success`, `status`, and `exitCode`; JUnit includes
a failing policy test case when thresholds fail.

Baseline format `2` binds FlowSpec and every referenced ServiceSpec by SHA-256.
Record a new baseline after a contract file changes or when migrating from format
`1`. `treat-as-absolute` handles a missing file only; malformed or incompatible
baselines remain errors. Threshold values must be finite numbers between 0 and 1.

Reports bind contract files, trace data, the consumed baseline, validator binary,
and validation settings under `inputs` (JUnit: `result.inputs` property).

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
