# FlowSpec Schema Reference

## Overview

The FlowSpec schema defines the structure for ChoreoAtlas flow specifications. It supports two formats:
- **Graph format** (recommended): DAG-based flow with explicit dependencies
- **Flow format** (legacy): Sequential step-based flow

## Schema Location

- **External Schema**: `schemas/flowspec.schema.json`
- **Internal Schema**: `internal/schemas/flowspec.schema.json`
- **Schema ID**: `https://github.com/choreoatlas2025/cli/schemas/flowspec.schema.json`

## Format Selection

The schema uses `oneOf` to enforce mutual exclusivity between graph and flow formats:
- A FlowSpec file must have either `graph` or `flow`, but not both
- Graph format is recommended for new projects
- Flow format is maintained for backward compatibility

## Graph Format (Recommended)

### Structure
```yaml
info:
  title: "Flow Title"
  description: "Optional description"
  version: "1.0.0"

services:
  serviceName:
    spec: "./path/to/service.spec.yaml"

graph:
  nodes:
    - id: "nodeId"
      call: "serviceName.operationId"
      depends: ["previousNodeId"]  # Optional: dependencies
      input:                       # Optional: input mappings
        param: "${variable}"
      output:                      # Optional: output mappings
        varName: "response.field"
  edges: []  # Optional: explicit edges (auto-generated from depends)
```

### Key Features
- **Nodes**: Each node represents a service operation call
- **Dependencies**: Use `depends` array to specify node dependencies
- **Edges**: Automatically generated from `depends` field
- **Variables**: Flow between nodes via `output` and `input` mappings

### Example
```yaml
info:
  title: "Order Processing"
  version: "1.0.0"

services:
  orderService:
    spec: "../services/order-service.servicespec.yaml"
  inventoryService:
    spec: "../services/inventory-service.servicespec.yaml"

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

## Flow Format (Legacy)

### Structure
```yaml
info:
  title: "Flow Title"

services:
  serviceName:
    spec: "./path/to/service.spec.yaml"

flow:
  - step: "Step Name"
    call: "serviceName.operationId"
    input:
      param: "${variable}"
    output:
      varName: "response.field"
```

### Parallel Steps
```yaml
flow:
  - step: "Parallel Group"
    parallel:
      - step: "Parallel Step 1"
        call: "service.operation1"
      - step: "Parallel Step 2"
        call: "service.operation2"
```

## Validation

The schema is validated at two levels:

1. **Structure Validation**: JSON Schema validation of the YAML structure
2. **Semantic Validation**: Business logic validation (service references, variable flow, etc.)

## VSCode Integration

See [VSCode Setup Guide](../schemas/vscode-setup.md) for auto-completion and validation support.

## Migration Guide

To migrate from flow to graph format:

1. Convert sequential steps to nodes with unique IDs
2. Add `depends` field to establish execution order
3. Keep the same `input`/`output` mappings
4. Test with `choreoatlas lint` to verify correctness

## Related Documentation

- [VSCode Setup](../schemas/vscode-setup.md)
- [Graph Format Guide](./graph-format.md)
- [Migration Guide](../migration/flow-to-graph.md)