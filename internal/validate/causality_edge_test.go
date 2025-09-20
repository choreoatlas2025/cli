package validate

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestValidateEdgeConstraints(t *testing.T) {
	tests := []struct {
		name           string
		spans          []trace.Span
		toleranceMs    int64
		expectViolations int
		violationTypes []string
	}{
		{
			name: "no violations - proper parent-child timing",
			spans: []trace.Span{
				{
					Name:       "parent",
					Service:    "serviceA",
					StartNanos: 1000000000,
					EndNanos:   2000000000,
					Attributes: map[string]any{
						"otlp.span_id": "span1",
					},
				},
				{
					Name:       "child",
					Service:    "serviceB",
					StartNanos: 1100000000,
					EndNanos:   1900000000,
					Attributes: map[string]any{
						"otlp.span_id":        "span2",
						"otlp.parent_span_id": "span1",
					},
				},
			},
			toleranceMs:      50,
			expectViolations: 0,
		},
		{
			name: "parent-child violation - child exceeds parent bounds",
			spans: []trace.Span{
				{
					Name:       "parent",
					Service:    "serviceA",
					StartNanos: 1000000000,
					EndNanos:   2000000000,
					Attributes: map[string]any{
						"otlp.span_id": "span1",
					},
				},
				{
					Name:       "child",
					Service:    "serviceB",
					StartNanos: 900000000, // starts before parent
					EndNanos:   2100000000, // ends after parent
					Attributes: map[string]any{
						"otlp.span_id":        "span2",
						"otlp.parent_span_id": "span1",
					},
				},
			},
			toleranceMs:      10, // 10ms tolerance
			expectViolations: 1,
			violationTypes:   []string{"parent-child"},
		},
		{
			name: "temporal violation - operations not in sequence",
			spans: []trace.Span{
				{
					Name:       "first",
					Service:    "serviceA",
					StartNanos: 1000000000,
					EndNanos:   2000000000,
					Attributes: map[string]any{
						"otlp.span_id": "span1",
					},
				},
				{
					Name:       "second",
					Service:    "serviceB",
					StartNanos: 1500000000, // starts before first ends
					EndNanos:   2500000000,
					Attributes: map[string]any{
						"otlp.span_id": "span2",
					},
				},
			},
			toleranceMs:      100, // 100ms tolerance
			expectViolations: 0,  // No explicit follows relationship, so no violation
			violationTypes:   nil,
		},
		{
			name: "concurrent operations without overlap",
			spans: []trace.Span{
				{
					Name:       "concurrent1",
					Service:    "serviceA",
					StartNanos: 1000000000,
					EndNanos:   1500000000,
					Attributes: map[string]any{
						"otlp.span_id": "span1",
					},
				},
				{
					Name:       "concurrent2",
					Service:    "serviceB",
					StartNanos: 1600000000, // no overlap
					EndNanos:   2000000000,
					Attributes: map[string]any{
						"otlp.span_id": "span2",
					},
				},
			},
			toleranceMs:      50,
			expectViolations: 0, // No violation without explicit concurrent relationship
		},
		{
			name: "cycle detection",
			spans: []trace.Span{
				{
					Name:       "nodeA",
					Service:    "serviceA",
					StartNanos: 1000000000,
					EndNanos:   2000000000,
					Attributes: map[string]any{
						"otlp.span_id":        "span1",
						"otlp.parent_span_id": "span3", // creates cycle
					},
				},
				{
					Name:       "nodeB",
					Service:    "serviceB",
					StartNanos: 1100000000,
					EndNanos:   1900000000,
					Attributes: map[string]any{
						"otlp.span_id":        "span2",
						"otlp.parent_span_id": "span1",
					},
				},
				{
					Name:       "nodeC",
					Service:    "serviceC",
					StartNanos: 1200000000,
					EndNanos:   1800000000,
					Attributes: map[string]any{
						"otlp.span_id":        "span3",
						"otlp.parent_span_id": "span2",
					},
				},
			},
			toleranceMs:      50,
			expectViolations: 2, // Both cycle and parent-child violation
			violationTypes:   []string{"cycle", "parent-child"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			graph, err := BuildCallGraph(tc.spans)
			if err != nil {
				t.Fatalf("Failed to build call graph: %v", err)
			}

			toleranceNanos := tc.toleranceMs * 1000000
			violations := graph.ValidateEdgeConstraints(toleranceNanos)

			if len(violations) != tc.expectViolations {
				t.Errorf("Expected %d violations, got %d", tc.expectViolations, len(violations))
				for _, v := range violations {
					t.Logf("  Violation: type=%s, message=%s", v.Type, v.Message)
				}
			}

			// Check violation types if specified
			if tc.violationTypes != nil {
				violationTypeMap := make(map[string]bool)
				for _, v := range violations {
					violationTypeMap[v.Type] = true
				}

				for _, expectedType := range tc.violationTypes {
					if !violationTypeMap[expectedType] {
						t.Errorf("Expected violation type '%s' not found", expectedType)
					}
				}
			}
		})
	}
}

func TestDetectCycle(t *testing.T) {
	tests := []struct {
		name      string
		spans     []trace.Span
		hasCycle  bool
		cycleSize int
	}{
		{
			name: "no cycle - linear chain",
			spans: []trace.Span{
				{
					Name:    "A",
					Service: "svc",
					Attributes: map[string]any{
						"otlp.span_id": "A",
					},
				},
				{
					Name:    "B",
					Service: "svc",
					Attributes: map[string]any{
						"otlp.span_id":        "B",
						"otlp.parent_span_id": "A",
					},
				},
				{
					Name:    "C",
					Service: "svc",
					Attributes: map[string]any{
						"otlp.span_id":        "C",
						"otlp.parent_span_id": "B",
					},
				},
			},
			hasCycle: false,
		},
		{
			name: "simple cycle - A->B->C->A",
			spans: []trace.Span{
				{
					Name:    "A",
					Service: "svc",
					Attributes: map[string]any{
						"otlp.span_id":        "A",
						"otlp.parent_span_id": "C",
					},
				},
				{
					Name:    "B",
					Service: "svc",
					Attributes: map[string]any{
						"otlp.span_id":        "B",
						"otlp.parent_span_id": "A",
					},
				},
				{
					Name:    "C",
					Service: "svc",
					Attributes: map[string]any{
						"otlp.span_id":        "C",
						"otlp.parent_span_id": "B",
					},
				},
			},
			hasCycle:  true,
			cycleSize: 4, // A->B->C->A
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			graph, err := BuildCallGraph(tc.spans)
			if err != nil {
				t.Fatalf("Failed to build call graph: %v", err)
			}

			hasCycle, cyclePath := graph.DetectCycle()

			if hasCycle != tc.hasCycle {
				t.Errorf("Expected hasCycle=%v, got %v", tc.hasCycle, hasCycle)
			}

			if hasCycle && tc.cycleSize > 0 {
				if len(cyclePath) != tc.cycleSize {
					t.Errorf("Expected cycle size %d, got %d: %v", tc.cycleSize, len(cyclePath), cyclePath)
				}
			}
		})
	}
}

func TestGetTopologicalOrder(t *testing.T) {
	spans := []trace.Span{
		{
			Name:    "A",
			Service: "svc",
			Attributes: map[string]any{
				"otlp.span_id": "A",
			},
		},
		{
			Name:    "B",
			Service: "svc",
			Attributes: map[string]any{
				"otlp.span_id":        "B",
				"otlp.parent_span_id": "A",
			},
		},
		{
			Name:    "C",
			Service: "svc",
			Attributes: map[string]any{
				"otlp.span_id":        "C",
				"otlp.parent_span_id": "A",
			},
		},
		{
			Name:    "D",
			Service: "svc",
			Attributes: map[string]any{
				"otlp.span_id":        "D",
				"otlp.parent_span_id": "B",
			},
		},
	}

	graph, err := BuildCallGraph(spans)
	if err != nil {
		t.Fatalf("Failed to build call graph: %v", err)
	}

	order, err := graph.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("Failed to get topological order: %v", err)
	}

	if len(order) != 4 {
		t.Errorf("Expected 4 nodes in order, got %d", len(order))
	}

	// Verify A comes before B and C
	indexA, indexB, indexC, indexD := -1, -1, -1, -1
	for i, id := range order {
		if id == "A" {
			indexA = i
		} else if id == "B" {
			indexB = i
		} else if id == "C" {
			indexC = i
		} else if id == "D" {
			indexD = i
		}
	}

	if indexA >= indexB || indexA >= indexC {
		t.Errorf("A should come before B and C in topological order")
	}
	if indexB >= indexD {
		t.Errorf("B should come before D in topological order")
	}
}