# GitHub Actions Integration

## Overview

ChoreoAtlas CLI integrates seamlessly with GitHub Actions for continuous validation of your service choreography contracts. This guide provides ready-to-use workflows for common CI/CD scenarios.

## Quick Start

### Minimal Workflow

Create `.github/workflows/choreoatlas.yml`:

```yaml
name: Validate Contracts
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Validate FlowSpec
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest lint \
            --flow /workspace/contracts/main.flowspec.yaml
```

## Complete Examples

### 1. Basic Validation

File: `.github/workflows/minimal.yml`

```yaml
name: Contract Validation
on:
  push:
    branches: [main]
    paths:
      - 'contracts/**'
      - 'traces/**'
  pull_request:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Lint contracts
        run: |
          docker run --rm -v $PWD:/workspace choreoatlas/cli:latest \
            lint --flow /workspace/contracts/main.flowspec.yaml

      - name: Validate against traces
        run: |
          docker run --rm -v $PWD:/workspace choreoatlas/cli:latest \
            validate \
            --flow /workspace/contracts/main.flowspec.yaml \
            --trace /workspace/traces/test.trace.json
```

### 2. Complete CI/CD Workflow

File: `.github/workflows/combo.yml`

```yaml
name: ChoreoAtlas CI/CD
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  workflow_dispatch:
    inputs:
      trace_file:
        description: 'Trace file to validate against'
        required: false
        default: 'traces/test.trace.json'

env:
  FLOWSPEC_PATH: contracts/main.flowspec.yaml
  TRACE_PATH: ${{ github.event.inputs.trace_file || 'traces/test.trace.json' }}

jobs:
  lint:
    name: Static Validation
    runs-on: ubuntu-latest

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Set up Docker
        uses: docker/setup-buildx-action@v3

      - name: Lint FlowSpec
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest lint \
            --flow /workspace/${{ env.FLOWSPEC_PATH }}

      - name: Upload FlowSpec
        uses: actions/upload-artifact@v3
        with:
          name: flowspec
          path: ${{ env.FLOWSPEC_PATH }}

  validate:
    name: Dynamic Validation
    runs-on: ubuntu-latest
    needs: lint

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Validate against trace
        id: validate
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest validate \
            --flow /workspace/${{ env.FLOWSPEC_PATH }} \
            --trace /workspace/${{ env.TRACE_PATH }} \
            --report-format json \
            --report-out /workspace/validation-report.json

      - name: Upload validation report
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: validation-report
          path: validation-report.json

      - name: Generate HTML report
        if: always()
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest validate \
            --flow /workspace/${{ env.FLOWSPEC_PATH }} \
            --trace /workspace/${{ env.TRACE_PATH }} \
            --report-format html \
            --report-out /workspace/validation-report.html

      - name: Upload HTML report
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: html-report
          path: validation-report.html

  gate:
    name: Quality Gate
    runs-on: ubuntu-latest
    needs: validate

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: CI Gate Check
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest ci-gate \
            --flow /workspace/${{ env.FLOWSPEC_PATH }} \
            --trace /workspace/${{ env.TRACE_PATH }}

      - name: Comment on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v6
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '✅ Contract validation passed!'
            })
```

### 3. Discovery Workflow

File: `.github/workflows/discover.yml`

```yaml
name: Contract Discovery
on:
  workflow_dispatch:
    inputs:
      trace_url:
        description: 'URL to download trace.json'
        required: true
      flow_title:
        description: 'Title for discovered flow'
        required: false
        default: 'Discovered Flow'

jobs:
  discover:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Download trace
        run: |
          curl -L "${{ github.event.inputs.trace_url }}" \
            -o trace-input.json

      - name: Discover contracts
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest discover \
            --trace /workspace/trace-input.json \
            --out /workspace/discovered.flowspec.yaml \
            --out-services /workspace/discovered-services \
            --title "${{ github.event.inputs.flow_title }}"

      - name: Validate discovered contracts
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest lint \
            --flow /workspace/discovered.flowspec.yaml

      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v5
        with:
          branch: discovered-contracts
          title: "Discovered contracts from trace"
          body: |
            ## Contract Discovery Results

            Generated from trace: ${{ github.event.inputs.trace_url }}

            ### Files Generated
            - FlowSpec: `discovered.flowspec.yaml`
            - ServiceSpecs: `discovered-services/`

            ### Next Steps
            1. Review the generated contracts
            2. Adjust variable mappings
            3. Add business rules and constraints
            4. Run validation tests
          commit-message: "feat: discovered contracts from trace"
```

### 4. Matrix Testing

File: `.github/workflows/matrix-test.yml`

```yaml
name: Matrix Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        flowspec:
          - contracts/order.flowspec.yaml
          - contracts/payment.flowspec.yaml
          - contracts/shipping.flowspec.yaml
        trace:
          - traces/success.trace.json
          - traces/failure.trace.json

    steps:
      - uses: actions/checkout@v4

      - name: Validate ${{ matrix.flowspec }} against ${{ matrix.trace }}
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest validate \
            --flow /workspace/${{ matrix.flowspec }} \
            --trace /workspace/${{ matrix.trace }}
        continue-on-error: true

      - name: Generate report
        if: always()
        run: |
          docker run --rm -v ${{ github.workspace }}:/workspace \
            choreoatlas/cli:latest validate \
            --flow /workspace/${{ matrix.flowspec }} \
            --trace /workspace/${{ matrix.trace }} \
            --report-format junit \
            --report-out /workspace/junit-${{ strategy.job-index }}.xml

      - name: Publish test results
        uses: dorny/test-reporter@v1
        if: always()
        with:
          name: Contract Validation Results
          path: 'junit-*.xml'
          reporter: java-junit
```

## Environment Variables

Configure ChoreoAtlas behavior using environment variables:

```yaml
env:
  # Default paths
  CHOREOATLAS_FLOW: contracts/main.flowspec.yaml
  CHOREOATLAS_TRACE: traces/test.trace.json

  # Validation options
  CHOREOATLAS_SKIP_SCHEMA: false
  CHOREOATLAS_SEMANTIC: true
  CHOREOATLAS_CAUSALITY: temporal

  # Thresholds
  CHOREOATLAS_THRESHOLD_STEPS: 0.9
  CHOREOATLAS_THRESHOLD_CONDS: 0.95
```

## Caching

Speed up workflows by caching Docker images:

```yaml
- name: Cache Docker layers
  uses: actions/cache@v3
  with:
    path: /tmp/.buildx-cache
    key: ${{ runner.os }}-buildx-${{ github.sha }}
    restore-keys: |
      ${{ runner.os }}-buildx-
```

## Exit Codes

ChoreoAtlas uses specific exit codes for different failure types:

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | Continue pipeline |
| 1 | CLI error | Check command syntax |
| 2 | Input error | Verify file paths and formats |
| 3 | Validation failed | Review contract violations |
| 4 | Gate failed | Thresholds not met |

Handle exit codes in workflows:

```yaml
- name: Validate with error handling
  id: validate
  run: |
    docker run --rm -v $PWD:/workspace choreoatlas/cli:latest \
      validate --flow /workspace/flow.yaml --trace /workspace/trace.json
  continue-on-error: true

- name: Handle validation failure
  if: steps.validate.outcome == 'failure'
  run: |
    case ${{ steps.validate.conclusion }} in
      3) echo "::error::Validation failed - check contract violations";;
      4) echo "::error::Quality gate failed - coverage below threshold";;
      *) echo "::error::Unexpected error occurred";;
    esac
    exit 1
```

## Best Practices

### 1. Use Specific Docker Tags

```yaml
# Pin to specific version
choreoatlas/cli:v0.7.0-ce

# Or use latest for development
choreoatlas/cli:latest
```

### 2. Separate Workflows

- **PR validation**: Lint + basic validation
- **Main branch**: Full validation + reports
- **Release**: Comprehensive testing + gates

### 3. Artifact Management

```yaml
- name: Save reports
  uses: actions/upload-artifact@v3
  with:
    name: validation-reports-${{ github.run_number }}
    path: |
      *.html
      *.json
      *.xml
    retention-days: 30
```

### 4. Notifications

```yaml
- name: Slack notification
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: ${{ job.status }}
    text: 'Contract validation failed for ${{ github.ref }}'
    webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

## Troubleshooting

### Docker Permission Issues

```yaml
- name: Fix permissions
  run: |
    sudo chown -R $USER:$USER ${{ github.workspace }}
```

### Path Mounting

Always use absolute paths in Docker:

```yaml
# Good
docker run -v ${{ github.workspace }}:/workspace

# Avoid
docker run -v .:/workspace
```

### Debugging

Enable debug output:

```yaml
- name: Debug validation
  run: |
    docker run --rm -v $PWD:/workspace \
      -e DEBUG=true \
      choreoatlas/cli:latest validate \
      --flow /workspace/flow.yaml \
      --trace /workspace/trace.json
```

## Examples Repository

Find complete working examples at:
https://github.com/choreoatlas2025/cli-examples

## Next Steps

- [Installation Guide](../installation.md) - Set up ChoreoAtlas locally
- [Discovery Guide](../discovery/from-trace.md) - Generate contracts from traces
- [Schema Reference](../flowspec/schema.md) - Contract structure details