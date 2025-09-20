# Causality and DAG Validation

## Overview

ChoreoAtlas CLI provides comprehensive causality checking and DAG (Directed Acyclic Graph) validation to ensure your distributed system's execution follows the expected flow patterns. This includes cycle detection, edge constraint validation, and temporal relationship verification.

## Features

### 1. DAG Cycle Detection
Automatically detects circular dependencies in your service call graph to prevent infinite loops and deadlocks.

### 2. Edge Constraint Validation
Validates three types of relationships with configurable time tolerance:

- **Parent-Child Relationships**: Ensures child spans execute within their parent's time bounds
- **Temporal Relationships**: Validates that predecessor operations complete before successors start
- **Concurrent Relationships**: Verifies that concurrent operations actually overlap in time

### 3. Topological Sorting
Generates a valid execution order for all operations respecting dependencies.

## Configuration

### Causality Mode
Control the level of causality checking with the `--causality` flag:

- `strict`: Use parent-child span relationships from OTLP data
- `temporal`: Use temporal ordering based on timestamps (default)
- `off`: Disable causality checking

### Time Tolerance
Configure time tolerance for edge constraints with `--causality-tolerance` (in milliseconds):

```bash
# Allow 100ms tolerance for timing variations
choreoatlas validate --flow order.flowspec.yaml \
  --trace trace.json \
  --causality-tolerance 100
```

Default tolerance is 50ms to account for clock skew and network latency.

## Validation Rules

### Parent-Child Constraints
For spans with parent-child relationships:
- Child start time must be ≥ parent start time - tolerance
- Child end time must be ≤ parent end time + tolerance

### Temporal Constraints
For sequential operations (A → B):
- A.endTime must be ≤ B.startTime + tolerance

### Concurrency Constraints
For parallel operations marked as concurrent:
- Time ranges must overlap: A.startTime < B.endTime AND B.startTime < A.endTime

## Example Violations

### Cycle Detection
```
[DAG Violation] cycle: Cycle detected: [orderService:createOrder → inventoryService:checkStock → orderService:updateOrder → inventoryService:checkStock]
```

### Temporal Violation
```
[DAG Violation] causality: Causality constraint violation: orderService.createOrder should complete before paymentService.processPayment (tolerance 50ms)
```

### Concurrency Violation
```
[DAG Violation] overlap: Concurrency constraint violation: inventoryService.reserve and pricingService.calculate should overlap but don't
```

## FlowSpec Examples

### Sequential Flow
```yaml
flow:
  - step: "Create Order"
    call: "orderService.createOrder"
    output:
      orderId: "response.id"

  - step: "Process Payment"
    call: "paymentService.process"
    input:
      orderId: "${orderId}"
```

### Concurrent Flow
```yaml
flow:
  - step: "Prepare Order"
    call: "orderService.prepare"

  - step: "Parallel Processing"
    parallel:
      - step: "Check Inventory"
        call: "inventoryService.check"
      - step: "Calculate Pricing"
        call: "pricingService.calculate"

  - step: "Finalize"
    call: "orderService.finalize"
```

## Graph Mode Support

DAG validation also works with graph-format FlowSpecs:

```yaml
graph:
  nodes:
    - id: "start"
      call: "orderService.create"
    - id: "inventory"
      call: "inventoryService.reserve"
      depends: ["start"]
    - id: "payment"
      call: "paymentService.process"
      depends: ["start"]
    - id: "ship"
      call: "shippingService.create"
      depends: ["inventory", "payment"]
```

## Best Practices

1. **Set Appropriate Tolerance**: Consider your system's network latency and clock synchronization when setting tolerance
2. **Use Semantic Versioning**: Include version info in your FlowSpec to track changes
3. **Test with Real Traces**: Validate against production traces to catch real-world timing issues
4. **Monitor Degradation**: Use baseline comparison to detect performance regression

## Troubleshooting

### Clock Skew Issues
If you see many false positive violations, increase tolerance:
```bash
--causality-tolerance 200  # 200ms tolerance
```

### Missing Parent Spans
Ensure your tracing captures complete call chains. Missing parent spans will cause validation failures.

### Concurrent vs Sequential
Mark operations as parallel only if they truly execute concurrently. False concurrency declarations will cause validation failures.

## Integration with CI/CD

Include causality validation in your CI pipeline:

```yaml
# GitHub Actions example
- name: Validate Flow Causality
  run: |
    choreoatlas validate \
      --flow flow.yaml \
      --trace test-trace.json \
      --causality strict \
      --causality-tolerance 100 \
      --threshold-steps 0.95
```

## Related Documentation

- [Baseline Comparison](../reports/ce-report.md) - Track validation quality over time
- [FlowSpec Schema](../schemas/flowspec.md) - Full FlowSpec specification
- [Trace Format](../traces/format.md) - Supported trace formats