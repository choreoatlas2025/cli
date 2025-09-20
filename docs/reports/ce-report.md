# Baseline and Report Documentation

## Overview

The ChoreoAtlas CLI supports baseline comparison for tracking performance and validation quality over time. When a baseline is provided, the system switches from absolute threshold checking to relative comparison mode.

## Baseline Comparison Modes

### Absolute Mode (No Baseline)
When no baseline is provided, thresholds are checked against absolute values:
- Step coverage must meet the specified `--threshold-steps` (default 90%)
- Condition pass rate must meet the specified `--threshold-conds` (default 95%)

### Relative Mode (With Baseline)
When a baseline is provided via `--baseline`, the system compares current metrics against the baseline:
- Calculates delta percentages for both step coverage and condition pass rates
- Thresholds now represent maximum allowed degradation
- Example: `--threshold-steps 0.1` allows up to 10% degradation from baseline

## Report Fields

### Standard Fields
- `stepsTotal`: Total number of steps in the flow
- `stepsPass`: Number of steps that passed validation
- `stepsCoverage`: Percentage of steps covered (0.0-1.0)
- `conditionsTotal`: Total number of conditions evaluated
- `conditionsPass`: Number of conditions that passed
- `conditionsRate`: Pass rate for conditions (0.0-1.0)

### Baseline Fields (when baseline is provided)
- `baselineStepsCoverage`: Step coverage from baseline
- `stepsDeltaAbs`: Absolute difference in step coverage
- `stepsDeltaPct`: Percentage change from baseline
- `baselineConditionsRate`: Condition pass rate from baseline
- `conditionsDeltaAbs`: Absolute difference in condition rate
- `conditionsDeltaPct`: Percentage change from baseline

## Example Output

### Without Baseline
```
[GATE] Baseline Gate: PASSED ✓
  Steps Coverage: 95.0% (>= 90.0%)
  Conditions Pass Rate: 98.0% (>= 95.0%)
```

### With Baseline
```
[GATE] Baseline Gate: PASSED ✓
  Steps Coverage: 93.0% (>= 90.0%)
  Conditions Pass Rate: 96.0% (>= 95.0%)
  Baseline Comparison:
    Steps: 95.0% baseline → 93.0% current (delta: -2.1%)
    Conditions: 98.0% baseline → 96.0% current (delta: -2.0%)
```

## Baseline Missing Strategy

The `--baseline-missing` flag controls behavior when the specified baseline file cannot be loaded:

- `fail` (default): Exit with an error if the baseline file cannot be loaded
- `treat-as-absolute`: Fall back to absolute threshold mode with a warning

## Usage Examples

### Record a Baseline
```bash
choreoatlas baseline record \
  --flow order-flow.flowspec.yaml \
  --trace traces/golden.json \
  --out baseline.json
```

### Validate with Baseline Comparison
```bash
choreoatlas validate \
  --flow order-flow.flowspec.yaml \
  --trace traces/current.json \
  --baseline baseline.json \
  --threshold-steps 0.05 \
  --threshold-conds 0.03
```

### Handle Missing Baseline
```bash
choreoatlas validate \
  --flow order-flow.flowspec.yaml \
  --trace traces/current.json \
  --baseline baseline.json \
  --baseline-missing treat-as-absolute
```