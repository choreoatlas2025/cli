# ChoreoAtlas CLI

**Map. Verify. Steer** cross-service choreography

ChoreoAtlas is a Contract-as-Code platform for interactive logic governance, following the "Discover-Specify-Guide" closed-loop concept. It supports dual contract mode (ServiceSpec & FlowSpec) and provides Atlas Scout (Discovery), Atlas Proof (Verification), and Atlas Pilot (Guidance) components.

## 🚀 Quick Start

### Installation

#### Docker (Recommended)
```bash
docker run --rm ghcr.io/choreoatlas2025/cli:latest --help
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
# Mount examples directory and run validation
docker run --rm -v $(pwd)/examples:/examples ghcr.io/choreoatlas2025/cli:latest lint --flow /examples/flows/order-fulfillment.flowspec.yaml

# Generate HTML report
docker run --rm -v $(pwd):/workspace ghcr.io/choreoatlas2025/cli:latest validate \
  --flow /workspace/examples/flows/order-fulfillment.flowspec.yaml \
  --trace /workspace/examples/traces/successful-order.trace.json \
  --report-format html --report-out /workspace/report.html
```

## ✨ Core Features

### Contract-as-Code Approach
- **FlowSpec**: Central choreography specification defining step sequences, service calls, and variable flow
- **ServiceSpec**: Per-service contracts with operation specifications, preconditions, and postconditions
- **Dual Validation**: Both static (lint) and dynamic (runtime trace) validation

### Atlas Components
- **Atlas Scout** (`discover`): Generate FlowSpec from trace exploration
- **Atlas Proof** (`validate`): Verify choreography matches actual execution
- **Atlas Pilot** (`lint`): Static validation and guidance

### Enterprise Features (Community Edition)
- **JSON Schema Validation**: Structured validation of FlowSpec and ServiceSpec formats
- **Multiple Report Formats**: JSON, JUnit XML, and HTML reports for seamless CI integration
- **Trace-based Discovery**: Semi-automatic FlowSpec generation from trace.json
- **Temporal Validation**: Timestamp-based step sequence verification
- **CI/CD Integration**: Non-zero exit codes for pipeline integration

## 📋 Contract Structure

### FlowSpec Format
```yaml
info:
  title: "Order Fulfillment Process"
  version: "1.0.0"
services:
  orderService:
    spec: "./services/order-service.servicespec.yaml"
  inventoryService:
    spec: "./services/inventory-service.servicespec.yaml"
flow:
  - step: "Create Order"
    call: "orderService.createOrder"
    input:
      customerId: "${customerId}"
      items: "${items}"
    output:
      orderResponse: "response.body"
  - step: "Reserve Inventory"
    call: "inventoryService.reserveInventory"
    input:
      orderId: "${orderResponse.orderId}"
      items: "${items}"
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
- `1`: General CLI error
- `2`: File not found or parsing error
- `3`: Validation failed (spec vs trace mismatch)

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

## 📦 Edition Features

| Edition | Features |
|---------|----------|
| **Community Edition** | Basic Lint + File-based Validate |
| **Pro-Free** | + OTLP Collection |
| **Pro-Privacy** | + PII Masking |
| **Cloud** | + Remote Storage & Collaboration |

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## 🔗 Links

- **Documentation**: https://choreoatlas.io (Coming Soon)
- **GitHub**: https://github.com/choreoatlas2025/cli
- **Docker**: https://github.com/choreoatlas2025/cli/pkgs/container/cli
- **Issues**: https://github.com/choreoatlas2025/cli/issues
- **Discussions**: https://github.com/choreoatlas2025/cli/discussions

---

*ChoreoAtlas CLI - Map, Verify, and Steer your service choreography with Contract-as-Code*