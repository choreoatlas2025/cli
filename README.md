# ChoreoAtlas CLI

[![Version](https://img.shields.io/github/v/tag/choreoatlas2025/cli?label=version)](https://github.com/choreoatlas2025/cli/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/docker/v/choreoatlas/cli?label=docker)](https://hub.docker.com/r/choreoatlas/cli)

**Map. Verify. Steer** cross-service choreography

> ⚠️ **Beta Release**: v0.2.0-ce.beta.1 - Community Edition with zero telemetry

ChoreoAtlas is a Contract-as-Code platform for interactive logic governance, following the "Discover-Specify-Guide" closed-loop concept. It supports dual contract mode (ServiceSpec & FlowSpec) and provides Atlas Scout (Discovery), Atlas Proof (Verification), and Atlas Pilot (Guidance) components.

## 🚀 Quick Start

### Installation

#### Docker (Recommended)
```bash
# Public access via Docker Hub (currently v0.1.5-ce, v0.2.0 coming soon)
docker run --rm choreoatlas/cli:latest --help

# Run with local files mounted
docker run --rm -v $(pwd):/workspace choreoatlas/cli:latest lint --flow /workspace/your.flowspec.yaml
```

#### Homebrew (Coming Soon)
```bash
brew tap choreoatlas2025/tap
brew install choreoatlas
```

#### Manual Download
Download the appropriate binary from our [releases page](https://github.com/choreoatlas2025/cli/releases) and add it to your PATH.

### Basic Usage

```bash
# Static validation (includes JSON Schema validation)
choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml

# Dynamic validation against trace data
choreoatlas validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json

# Generate reports (JSON, JUnit, HTML)
choreoatlas validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --report-format html --report-out report.html

# Discover FlowSpec from trace data
choreoatlas discover --trace examples/traces/successful-order.trace.json --out discovered.yaml

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

### FlowSpec Format (Graph - Recommended)
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

### FlowSpec Format (Sequential - Legacy)
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

*ChoreoAtlas CLI - Map, Verify, and Steer your service choreography with Contract-as-Code*