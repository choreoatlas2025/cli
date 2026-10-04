# Baseline and Report Documentation

## Final result

Runtime validation and threshold policy have separate results. A failed step or
evaluated CEL condition fails runtime validation with exit code `3`. Thresholds
cannot override that failure. When runtime validation passes but the policy fails,
the command exits `4`. Only both passing permits exit `0`.

JSON and HTML expose the same final `success`, `status`, and `exitCode`, together
with `validationPassed` and `gatePassed`. JUnit represents a failed policy as a
failing test case, even when all runtime steps passed. Its `result.exitCode`
property records the command decision. Coverage statistics remain diagnostics.

## Threshold policy

Absolute floors apply with or without a baseline:

- `--threshold-steps`: minimum step coverage, default `0.9`.
- `--threshold-conds`: minimum condition pass rate, default `0.95`.

With a baseline, additional limits apply:

- `--max-steps-degradation`: maximum relative coverage degradation, default `0`.
- `--max-conds-degradation`: maximum relative condition rate degradation, default `0`.

Relative degradation is `(baselineRate - currentRate) / baselineRate` for a
positive baseline rate. The additional limits never replace absolute floors.
All four values must be finite numbers between `0` and `1`. A step-level failure
still fails runtime validation regardless of these settings.

```bash
choreoatlas validate \
  --flow order-flow.flowspec.yaml \
  --trace traces/current.json \
  --baseline baseline.json \
  --threshold-steps 0.9 \
  --threshold-conds 0.95 \
  --max-steps-degradation 0.05 \
  --max-conds-degradation 0.03
```

## Baseline identity

`baseline record` accepts complete successful results only. Failed, missing,
duplicate, or skipped condition results cannot be recorded as a successful
baseline. Actual calls are counted, including parallel children; containers
without a call do not inflate the count.

Format `2` records the FlowSpec title and SHA-256, every referenced ServiceSpec's
SHA-256, covered step identities, and evaluated condition identities. Loading and
comparison validate this data against the current contract. Changes to any bound
contract file, including formatting changes, require recording a new baseline.
Format `1` lacks complete service identity and must be replaced by a new recording.

```bash
choreoatlas baseline record \
  --flow order-flow.flowspec.yaml \
  --trace traces/golden.json \
  --out baseline.json
```

`--baseline-missing fail` is the default. `treat-as-absolute` falls back only when
the file does not exist; corrupt, unsupported, or incompatible baselines fail.

## Report input binding

Reports include the hashes of the FlowSpec, referenced ServiceSpecs, trace,
consumed baseline, and validator binary, plus validator version, semantic and
causality settings, and threshold policy. JSON and HTML use `inputs`; JUnit uses
the JSON-valued `result.inputs` property. These bindings identify which inputs and
rules produced a result; reports do not re-evaluate themselves after files change.

Baseline comparison details include `baselineStepsCoverage`, `stepsDeltaAbs`,
`stepsDeltaPct`, `baselineConditionsRate`, `conditionsDeltaAbs`, and
`conditionsDeltaPct`. Rate values are fractions between `0` and `1`; relative
changes can exceed that interval when a rate improves from a small baseline.
