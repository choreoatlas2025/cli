# ChoreoAtlas CLI

[![Release](https://img.shields.io/github/v/release/choreoatlas2025/cli?display_name=tag&include_prereleases&label=release)](https://github.com/choreoatlas2025/cli/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/docker/v/choreoatlas/cli?label=docker)](https://hub.docker.com/r/choreoatlas/cli)

Map. Verify. Steer your cross-service choreography — a developer-friendly, Swiss‑army‑knife style CLI for Contract‑as‑Code.

This is the Community Edition (CE): zero telemetry, fully offline.

ChoreoAtlas is a Contract‑as‑Code platform for interactive logic governance following the Discover → Specify → Guide loop. It supports dual contracts (FlowSpec + ServiceSpec) and provides:
- Atlas Scout: discover specs from trace data
- Atlas Proof: validate choreography against runtime behavior
- Atlas Pilot: static linting and guidance

Whitepaper: see `../whitepaper/ChoreoAtlas-Whitepaper-1.1.zh-CN.md`

Looking for Chinese? See README.zh-CN.md

## 🚀 Quick Start

### Installation

#### Homebrew (macOS & Linux)
```bash
brew install choreoatlas2025/homebrew-choreoatlas/choreoatlas
brew upgrade choreoatlas2025/homebrew-choreoatlas/choreoatlas  # 更新到最新版
```

The formula installs both `choreoatlas` and a `ca` helper symlink.

#### Shell installer (macOS/Linux)
```bash
curl -fsSL https://raw.githubusercontent.com/choreoatlas2025/cli/main/scripts/install.sh -o choreoatlas-install.sh
chmod +x choreoatlas-install.sh
./choreoatlas-install.sh            # 自动选择 /opt/homebrew/bin 或 /usr/local/bin
# Flags: --version vX.Y.Z-ce, --force, --no-symlink
```

#### PowerShell installer (Windows)
```powershell
Invoke-WebRequest https://raw.githubusercontent.com/choreoatlas2025/cli/main/scripts/install.ps1 -OutFile choreoatlas-install.ps1
pwsh ./choreoatlas-install.ps1      # 支持 -Version, -Force, -NoSymlink
```

#### Docker / GHCR
```bash
# Pin to a specific release tag
docker run --rm choreoatlas/cli:vX.Y.Z-ce version
docker run --rm ghcr.io/choreoatlas2025/cli:vX.Y.Z-ce version

# Or track the moving latest tag
docker run --rm choreoatlas/cli:latest version
docker run --rm ghcr.io/choreoatlas2025/cli:latest version
```

#### Manual download
1. Download the asset matching your OS/arch from [GitHub Releases](https://github.com/choreoatlas2025/cli/releases) (pattern: `choreoatlas_vX.Y.Z-ce_<os>_<arch>.tar.gz|zip`).
2. Download `SHA256SUMS.txt` from the same release and verify the archive.
3. Extract the archive and move `choreoatlas` into a directory on your PATH (`/opt/homebrew/bin`, `/usr/local/bin`, or `%LOCALAPPDATA%\ChoreoAtlas\bin`).
4. Optionally create your own alias/symlink if you skipped the installers: `ln -s choreoatlas /usr/local/bin/ca`.

Run `choreoatlas version` after installation to confirm the build and edition suffix (`-ce`).

### Bootstrap A Workspace

```bash
choreoatlas init
choreoatlas lint
choreoatlas validate --trace traces/successful-order.trace.json
```

- `init` generates FlowSpec, ServiceSpec, starter trace files, and optional GitHub Actions workflow in the current directory.
- Pass `--trace your-trace.json` to bootstrap from an existing trace.
- Use `--ci minimal|combo` to inject `.github/workflows/choreoatlas.yml`.

### Basic Usage

```bash
# Bootstrap an interactive starter project
choreoatlas init

# Static validation (includes JSON Schema validation)
choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml

# Dynamic validation against trace data
choreoatlas validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json

# Generate reports (JSON, JUnit, HTML)
choreoatlas validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --report-format html --report-out report.html

# Discover dual contracts from trace data (FlowSpec + ServiceSpec files)
choreoatlas discover \
  --trace examples/traces/successful-order.trace.json \
  --out discovered.flowspec.yaml \
  --out-services ./services
# By default, discover enforces JSON Schema + lint gates and only writes on success.
# To bypass (not recommended), add: --no-validate

# CI gate mode (combines lint + validate with proper exit codes)
choreoatlas ci-gate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json
```

### Docker Usage Examples

```bash
# Mount examples directory and run validation (using Docker Hub)
docker run --rm -v $(pwd)/examples:/examples choreoatlas/cli:latest lint --flow /examples/flows/order-fulfillment.flowspec.yaml

# Generate HTML report (using Docker Hub)
docker run --rm -v $(pwd):/workspace choreoatlas/cli:latest validate \
  --flow /workspace/examples/flows/order-fulfillment.flowspec.yaml \
  --trace /workspace/examples/traces/successful-order.trace.json \
  --report-format html --report-out /workspace/report.html
```

### TL;DR Cheat Sheet

Common one‑liners for day‑to‑day work:

```bash
# Create a starter workspace (FlowSpec + ServiceSpec + trace)
ca init

# Lint your FlowSpec (uses embedded JSON Schemas)
ca lint --flow .flowspec.yaml

# Validate against a trace with semantic checks and temporal causality
ca validate --flow .flowspec.yaml --trace trace.json

# Tighten the gate: require 100% steps and 100% conditions
ca validate --flow .flowspec.yaml --trace trace.json \
  --threshold-steps 1.0 --threshold-conds 1.0 --skip-as-fail

# Record a baseline from current run
ca baseline record --flow .flowspec.yaml --trace trace.json --out baseline.json

# Gate with thresholds and an existing baseline; tolerate missing baseline by using absolute thresholds
ca validate --flow .flowspec.yaml --trace trace.json \
  --baseline ci/baseline.json --baseline-missing treat-as-absolute \
  --threshold-steps 0.9 --threshold-conds 0.95

# Generate an HTML report for humans (also supports json|junit)
ca validate --flow .flowspec.yaml --trace trace.json \
  --report-format html --report-out report.html

# Discover specs from a trace (FlowSpec + ServiceSpec files)
ca discover --trace trace.json --out discovered.flowspec.yaml --out-services ./services
# Gate is enabled by default; use --no-validate to bypass (not recommended)

# Run lint + validate in one go (CI gate)
ca ci-gate --flow .flowspec.yaml --trace trace.json

# Run validation on all traces in a folder
for f in traces/*.json; do ca validate --flow .flowspec.yaml --trace "$f"; done
```

## 📦 Versions & Distribution

- **Tags & releases**: all binaries ship as `vX.Y.Z-ce`; the suffix is visible in `choreoatlas version` output.
- **Installers & Homebrew**: both pull the same release artifacts and create the optional `ca` helper when safe.
- **Containers**: multi-arch manifests are published to both Docker Hub (`choreoatlas/cli`) and GHCR (`ghcr.io/choreoatlas2025/cli`) with matching `vX.Y.Z-ce` and `latest` tags.
- **Checksums**: every release includes `SHA256SUMS.txt` for integrity verification.
- **Privacy**: Community Edition is permanently zero-telemetry — see `docs/privacy.md` for verification steps.

## 🔒 Community Edition (CE) Features

### Zero Telemetry Guarantee
- **No Data Collection**: Absolutely no telemetry, analytics, or usage tracking
- **Completely Offline**: Works without any network connections
- **Privacy First**: Your contracts and traces never leave your machine
- **Verifiable**: Check with `strings choreoatlas | grep telemetry` (returns nothing)

## ✨ Core Features

### Contract-as-Code Approach
- **FlowSpec**: Central choreography specification defining step sequences, service calls, and variable flow
- **ServiceSpec**: Per-service contracts with operation specifications, preconditions, and postconditions
- **Dual Validation**: Both static (lint) and dynamic (runtime trace) validation

### Atlas Components
- **Atlas Scout** (`discover`): Generate FlowSpec from trace exploration
- **Atlas Proof** (`validate`): Verify choreography matches actual execution
- **Atlas Pilot** (`lint`): Static validation and guidance

### Validation Features
- **JSON Schema Validation**: Structured validation of FlowSpec and ServiceSpec formats
- **Multiple Report Formats**: JSON, JUnit XML, and HTML reports with CE badge
- **Trace-based Discovery**: Automatic dual contract generation from trace.json
- **Temporal & Causal Validation**: Step sequence and dependency verification
- **CI/CD Integration**: Standardized exit codes for pipeline integration
- **Baseline Gating**: Coverage thresholds and condition pass rates

## 📋 Contract Structure

### FlowSpec Format (Graph - recommended)
```yaml
info:
  title: "Order Fulfillment Process"
  version: "1.0.0"
services:
  orderService:
    spec: "./services/order-service.servicespec.yaml"
  inventoryService:
    spec: "./services/inventory-service.servicespec.yaml"
graph:
  nodes:
    - id: "createOrder"
      call: "orderService.createOrder"
      output:
        orderId: "response.orderId"
    - id: "checkInventory"
      call: "inventoryService.reserveInventory"
      depends: ["createOrder"]
      input:
        orderId: "${orderId}"
      output:
        reservationId: "response.reservationId"
```

### FlowSpec Format (Sequential - legacy)
```yaml
info:
  title: "Order Fulfillment Process"
services:
  orderService:
    spec: "./services/order-service.servicespec.yaml"
flow:
  - step: "Create Order"
    call: "orderService.createOrder"
    output:
      orderId: "response.orderId"
  - step: "Reserve Inventory"
    call: "inventoryService.reserveInventory"
    input:
      orderId: "${orderId}"
```

### ServiceSpec Format
```yaml
service: "OrderService"
version: "1.0.0"
operations:
  - operationId: "createOrder"
    description: "Create a new order"
    preconditions:
      "validCustomer": "has(input.customerId) && input.customerId != ''"
      "hasItems": "size(input.items) > 0"
    postconditions:
      "orderCreated": "has(response.body.orderId)"
      "statusOk": "response.status == 200"
```

## 🎯 Examples

The project includes a complete "Order-Inventory-Fulfillment" e-commerce flow example:

- `examples/flows/order-fulfillment.flowspec.yaml` - Main flow specification
- `examples/services/` - Service contract specifications
- `examples/traces/` - Success and failure scenario trace data

Try the examples:
```bash
# Clone the repository
git clone https://github.com/choreoatlas2025/cli.git
cd cli

# Run the example
choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml
choreoatlas validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json
```

## 🧰 CLI Reference

Commands and important flags (defaults shown where relevant):

```text
choreoatlas init
  --mode string          Bootstrap mode: template|trace
  --trace string         trace.json path for from-trace mode
  --ci string            GitHub Actions workflow: none|minimal|combo
  --examples             Copy examples/* starter assets
  --yes                  Accept defaults without prompts
  --force                Overwrite existing files
  --out string           Target directory (default ".")
  --title string         Override FlowSpec title

choreoatlas lint
  --flow string          FlowSpec file path (default ".flowspec.yaml")
  --schema               Enable JSON Schema strict validation (default true)

choreoatlas validate
  --flow string          FlowSpec file path (default ".flowspec.yaml")
  --trace string         trace.json file path (required)
  --semantic bool        Enable semantic validation (CEL) (default true)
  --causality string     Causality mode: strict|temporal|off (default "temporal")
  --causality-tolerance int  Causality tolerance in ms (default 50)
  --baseline string      Baseline file path (optional)
  --baseline-missing string  Strategy when baseline missing: fail|treat-as-absolute (default "fail")
  --threshold-steps float    Step coverage threshold (default 0.9)
  --threshold-conds float    Condition pass threshold (default 0.95)
  --skip-as-fail        Treat SKIP conditions as FAIL
  --report-format string Report format: json|junit|html (optional)
  --report-out string    Report output path (required when using --report-format)

choreoatlas discover
  --trace string         trace.json file path (required)
  --out string           FlowSpec output (default "discovered.flowspec.yaml")
  --out-services string  ServiceSpec output directory (default "./services")
  --title string         FlowSpec title

choreoatlas ci-gate
  --flow string          FlowSpec file path
  --trace string         trace.json file path

choreoatlas baseline record
  --flow string          FlowSpec file path (default ".flowspec.yaml")
  --trace string         trace.json file path (required)
  --out string           Baseline output file (default "baseline.json")
```

Notes:
- Default FlowSpec is `.flowspec.yaml` in the current directory.
- ServiceSpec paths in `services.*.spec` are resolved relative to the FlowSpec file.
- Graph (DAG) format is recommended; legacy `flow:` remains supported.

## 🔧 CI/CD Integration

### GitHub Actions
```yaml
- name: Validate Service Choreography
  run: |
    choreoatlas ci-gate \
      --flow specs/main-flow.flowspec.yaml \
      --trace traces/integration-test.trace.json
```

### Exit Codes
- `0`: All validations passed
- `1`: General CLI error (invalid arguments, etc.)
- `2`: File not found or parsing error
- `3`: Validation failed (spec vs trace mismatch)
- `4`: Gate policy violations (threshold not met)

### Report Formats
- **JSON**: Structured data for programmatic processing
- **JUnit XML**: Direct CI/CD integration
- **HTML**: Human-readable reports with timeline visualization

## 🧪 Trace Input Format

CE expects a simple JSON file shaped like:

```json
{
  "spans": [
    {
      "name": "createOrder",
      "service": "orderService",
      "startNanos": 1693910000000000000,
      "endNanos": 1693910000100000000,
      "attributes": {"response.status": 201}
    }
  ]
}
```

Causality checks can use temporal ordering by default, or strict parent/child when your attributes include OTLP‑style `otlp.parent_span_id` and `otlp.span_id` fields.

## 🧩 Common Workflows

1) Start from a trace → discover contracts → refine → validate
```bash
ca discover --trace traces/happy.json --out flow.flowspec.yaml --out-services ./services
# Edit and refine the generated FlowSpec/ServiceSpec files
ca lint --flow flow.flowspec.yaml
ca validate --flow flow.flowspec.yaml --trace traces/happy.json --report-format html --report-out report.html
```

2) Establish a baseline and add a gate
```bash
ca baseline record --flow flow.flowspec.yaml --trace traces/happy.json --out ci/baseline.json
ca validate --flow flow.flowspec.yaml --trace traces/regression.json \
  --baseline ci/baseline.json --threshold-steps 0.9 --threshold-conds 0.95
```

3) Batch validate
```bash
for f in traces/*.json; do ca validate --flow flow.flowspec.yaml --trace "$f"; done
```

## 🧱 Troubleshooting

- “flowspec cannot have both 'graph' and 'flow' fields”: choose one format.
- “no matching span found in trace”: verify `service.operation` matches `FlowSpec` and trace ordering/causality settings.
- “DAG structure validation failed”: fix cycles, missing nodes, or unreachable nodes in `graph`.
- Baseline file missing: add `--baseline-missing treat-as-absolute` to rely on thresholds only.
- Relative ServiceSpec paths: they are resolved relative to the FlowSpec file location.

## 🏗️ Development

### Building from Source

```bash
# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linting
make lint

# Clean build artifacts
make clean
```

### Project Structure
```
.
├── cmd/choreoatlas/          # CLI entry point
├── internal/
│   ├── cli/                  # Command line processing
│   ├── spec/                 # Specification loading and parsing
│   ├── validate/             # Static and dynamic validation logic
│   ├── trace/                # Trace data processing
│   └── report/               # Report generation
├── examples/                 # Example files
│   ├── flows/               # FlowSpec examples
│   ├── services/            # ServiceSpec examples
│   └── traces/              # Trace data examples
└── schemas/                 # JSON Schema definitions
```

## 📦 Edition Comparison

| Feature | Community (CE) | Pro-Free | Pro-Privacy | Cloud |
|---------|---------------|----------|-------------|-------|
| **Core Validation** | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| **Discovery (Atlas Scout)** | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| **HTML/JSON/JUnit Reports** | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| **Telemetry** | ❌ Never | ⚠️ Optional | ❌ Never | ✅ Required |
| **OTLP Import** | ❌ | ✅ | ✅ | ✅ |
| **PII Masking** | ❌ | ❌ | ✅ | ✅ |
| **Advanced Baseline** | ❌ | ✅ | ✅ | ✅ |
| **Team Collaboration** | ❌ | ❌ | ❌ | ✅ |
| **Price** | Free Forever | $19/user/mo | $39/user/mo | Custom |

## 📋 System Requirements

- **Operating Systems**: Linux, macOS, Windows
- **Architecture**: amd64, arm64
- **For Building**: Go 1.21+ required
- **Disk Space**: ~50MB for binary
- **Network**: Not required (fully offline capable)

## 🤝 Contributing

We welcome contributions! Please check our [Issues](https://github.com/choreoatlas2025/cli/issues) page for areas where you can help.

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## 🔗 Links

- **GitHub Repository**: https://github.com/choreoatlas2025/cli
- **Releases**: https://github.com/choreoatlas2025/cli/releases
- **Docker Hub**: https://hub.docker.com/r/choreoatlas/cli
- **Issues & Feature Requests**: https://github.com/choreoatlas2025/cli/issues
- **Discussions**: https://github.com/choreoatlas2025/cli/discussions

---

*ChoreoAtlas CLI — Map, Verify, and Steer your service choreography with Contract‑as‑Code*
