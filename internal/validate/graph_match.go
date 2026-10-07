// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/verdict"
	"sort"
	"strings"
)

// matchGraphSteps only chooses call instances and checks declared relationships.
// It cannot evaluate conditions or publish output variables.
func matchGraphSteps(fs *spec.FlowSpec, calls *CallGraph, config spec.ValidationConfig, ctx context.Context) []matchedStep {
	var matches []matchedStep
	if err := fs.Graph.ValidateGraphStructure(); err != nil {
		return []matchedStep{{result: StepResult{Step: "graph-structure", Call: "internal", Status: "FAIL", Issue: verdict.InvalidContract, Message: err.Error()}}}
	}
	// Build span matching index by service.operation
	spanIndex := make(map[string][]trace.Span)
	for _, call := range calls.Nodes {
		span := call.Span
		canonical := normalize(spec.ComputeOperationID(span))
		spanIndex[normalize(span.Service)+"."+canonical] = append(spanIndex[normalize(span.Service)+"."+canonical], span)
		if raw := normalize(span.Name); raw != canonical && spec.OperationMatches(span.Name, span) {
			key := normalize(span.Service) + "." + raw
			spanIndex[key] = append(spanIndex[key], span)
		}
	}

	// Validate each node in topological order
	topOrder, err := topologicalSort(fs.Graph)
	if err != nil {
		// Should not happen if lint passed, but handle gracefully
		for _, node := range fs.Graph.Nodes {
			matches = append(matches, matchedStep{result: StepResult{
				Step:    node.ID,
				Call:    node.Call,
				Status:  "FAIL",
				Message: fmt.Sprintf("DAG topological sort failed: %v", err), Issue: verdict.InvalidContract,
			}})
		}
		return matches
	}

	// Match repeated calls in time order, independent of trace file order.
	for key := range spanIndex {
		sort.Slice(spanIndex[key], func(i, j int) bool {
			a, b := spanIndex[key][i], spanIndex[key][j]
			if a.StartNanos != b.StartNanos {
				return a.StartNanos < b.StartNanos
			}
			if a.EndNanos != b.EndNanos {
				return a.EndNanos < b.EndNanos
			}
			return getSpanID(a) < getSpanID(b)
		})
	}
	matchedSpans := map[string]*trace.Span{}
	// Track matched spans to avoid double-matching
	usedSpans := make(map[string]bool) // span service:name:startNanos

	for _, nodeID := range topOrder {
		if ctx.Err() != nil {
			return matches
		}
		node := findNodeByID(fs.Graph, nodeID)
		if node == nil {
			matches = append(matches, matchedStep{result: StepResult{
				Step:    nodeID,
				Call:    "",
				Status:  "FAIL",
				Message: "Node not found", Issue: verdict.InvalidContract,
			}})
			continue
		}

		// Find matching spans for this node
		candidateSpans := spanIndex[normalize(node.Call)]
		var matchedSpan *trace.Span

		var causalityErr error
		for i := range candidateSpans {
			span := &candidateSpans[i]
			if usedSpans[getSpanID(*span)] {
				continue
			}
			if err := validateCausality(node, span, fs.Graph, matchedSpans, config); err != nil {
				causalityErr = err
				continue
			}
			matchedSpan = span
			break
		}
		if matchedSpan == nil {
			message := "no unused matching span found in trace"
			if causalityErr != nil {
				message = causalityErr.Error()
			}
			matches = append(matches, matchedStep{result: StepResult{Step: node.ID, Call: node.Call, Status: "FAIL", Message: message, Issue: verdict.StructureMismatch}})
			continue
		}

		usedSpans[getSpanID(*matchedSpan)] = true
		matchedSpans[node.ID] = matchedSpan

		step := spec.FlowStep{Step: node.ID, Call: node.Call, Input: node.Input, Output: node.Output, Meta: node.Meta}
		result := StepResult{Step: node.ID, Call: node.Call, Status: "PASS"}
		matches = append(matches, matchedStep{step: step, node: calls.Nodes[getSpanID(*matchedSpan)], result: result})
	}

	return matches
}

// validateCausality checks causality constraints for DAG nodes
func validateCausality(node *spec.GraphNode, nodeSpan *trace.Span, graph *spec.GraphSpec, matchedSpans map[string]*trace.Span, config spec.ValidationConfig) error {
	for _, edge := range graph.Edges {
		if edge.To != node.ID || (config.Causality == "off" && edge.Relationship != "concurrent") {
			continue
		}
		predID := edge.From
		predSpan := matchedSpans[predID]
		if predSpan == nil {
			return fmt.Errorf("predecessor %s has no matched span", predID)
		}

		relationship := edge.Relationship
		if relationship == "" {
			if config.Causality == "strict" {
				relationship = "parent"
			} else {
				relationship = "follows"
			}
		}
		switch relationship {
		case "parent":
			// Check parent-child relationship
			if !isParentChild(predSpan, nodeSpan) {
				return fmt.Errorf("node %s should be child of %s (strict mode)", node.ID, predID)
			}
		case "follows":
			if !completesBefore(predSpan.EndNanos, nodeSpan.StartNanos, config.ToleranceMs) {
				return fmt.Errorf("node %s starts before predecessor %s completes (temporal mode)", node.ID, predID)
			}
		case "concurrent":
			if predSpan.StartNanos >= nodeSpan.EndNanos || nodeSpan.StartNanos >= predSpan.EndNanos {
				return fmt.Errorf("node %s must overlap predecessor %s (concurrent relationship)", node.ID, predID)
			}
		}
	}

	return nil
}

// Compare a nonnegative time difference without adding tolerance to a timestamp.
func completesBefore(end, start, toleranceMs int64) bool {
	return end <= start || end-start <= toleranceMs*1000000
}

// Helper functions
func topologicalSort(graph *spec.GraphSpec) ([]string, error) {
	// Build adjacency list and in-degree map
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all nodes
	for _, node := range graph.Nodes {
		inDegree[node.ID] = 0
	}

	// Build adjacency list and calculate in-degrees
	for _, edge := range graph.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
		inDegree[edge.To]++
	}

	// Kahn's algorithm
	var queue []string
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	sort.Strings(queue)
	var result []string
	for len(queue) > 0 {
		sort.Strings(queue)
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, neighbor := range adj[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(graph.Nodes) {
		return nil, fmt.Errorf("cycle detected in graph")
	}

	return result, nil
}

func findNodeByID(graph *spec.GraphSpec, id string) *spec.GraphNode {
	for i := range graph.Nodes {
		if graph.Nodes[i].ID == id {
			return &graph.Nodes[i]
		}
	}
	return nil
}

func getPredecessors(nodeID string, graph *spec.GraphSpec) []string {
	var preds []string
	for _, edge := range graph.Edges {
		if edge.To == nodeID && edge.CarriesOutputs() {
			preds = append(preds, edge.From)
		}
	}
	return preds
}

func isParentChild(parent, child *trace.Span) bool {
	parentID := getParentSpanID(*child)
	return parentID != "" && parentID == getSpanID(*parent)
}

// normalize 标准化字符串用于比较
func normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
