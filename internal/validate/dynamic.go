// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"fmt"
	"github.com/choreoatlas2025/cli/internal/verdict"
	"sort"
	"strings"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

// StepResult is the shared, renderer-independent validation record.
type StepResult = verdict.StepResult

// CausalityMode represents the causality checking mode
type CausalityMode string

const (
	CausalityStrict   CausalityMode = "strict"   // Use parent-child span relationships
	CausalityTemporal CausalityMode = "temporal" // Use temporal ordering
	CausalityOff      CausalityMode = "off"      // Disable causality checking
)

// ValidateAgainstTrace 根据追踪数据验证流程执行（支持因果和并发校验）
func ValidateAgainstTrace(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace, config spec.ValidationConfig) ([]StepResult, bool) {
	_, results, passed := ValidateWithPlan(fs, opIndex, tr, config)
	return results, passed
}

func ValidateWithPlan(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace, config spec.ValidationConfig) (*ContractPlan, []StepResult, bool) {
	plan, err := CompilePlan(fs, opIndex, config)
	if err != nil {
		return nil, []StepResult{{Step: "contract-plan", Call: "internal", Status: "FAIL", Message: err.Error()}}, false
	}
	results, passed := plan.ValidateTrace(tr)
	return plan, results, passed
}

func validateAgainstTrace(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace, config spec.ValidationConfig, eval *evaluation) ([]StepResult, bool) {
	if err := config.Validate(); err != nil {
		return []StepResult{{Step: "validation-config", Call: "internal", Status: "FAIL", Message: err.Error()}}, false
	}
	// Normalize a private graph copy; concurrent validations never mutate the caller's contract.
	if fs.Graph != nil {
		flowCopy, graphCopy := *fs, *fs.Graph
		graphCopy.Edges = append([]spec.GraphEdge(nil), fs.Graph.Edges...)
		flowCopy.Graph = &graphCopy
		fs = &flowCopy
	}
	if err := trace.ValidateTimestamps(tr.Spans, CausalityMode(config.Causality) != CausalityOff || hasParallel(fs.Flow) || hasConcurrentEdges(fs.Graph)); err != nil {
		return []StepResult{{Step: "trace-time", Call: "internal", Status: "FAIL", Message: err.Error()}}, false
	}
	if fs.IsGraphMode() {
		return validateGraphAgainstTrace(fs, opIndex, tr, config, eval)
	}
	return validateWithCausality(fs, opIndex, tr, config, eval)
}

func hasConcurrentEdges(graph *spec.GraphSpec) bool {
	if graph != nil {
		for _, edge := range graph.Edges {
			if edge.Relationship == "concurrent" {
				return true
			}
		}
	}
	return false
}

func hasParallel(steps []spec.FlowStep) bool {
	for _, step := range steps {
		if len(step.Parallel) > 0 {
			return true
		}
	}
	return false
}

// validateWithCausality 使用因果校验（支持并发）
func validateWithCausality(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace, config spec.ValidationConfig, eval *evaluation) ([]StepResult, bool) {
	// 构建调用图
	graph, err := BuildCallGraph(tr.Spans)
	if err != nil {
		return []StepResult{{
			Step:    "graph-build",
			Call:    "internal",
			Status:  "FAIL",
			Message: fmt.Sprintf("Failed to build call graph: %v", err),
		}}, false
	}

	// 验证DAG约束（循环检测、边约束等）
	toleranceNanos := config.ToleranceMs * 1000000 // 转换为纳秒
	var violations []EdgeViolation
	if CausalityMode(config.Causality) != CausalityOff {
		violations = graph.ValidateEdgeConstraints(toleranceNanos)
	}

	// 执行因果校验
	matched := matchFlowSteps(fs, graph, config)
	results := evaluateFlowMatches(matched, opIndex, config, eval)
	allPassed := AllStepsPassed(results)

	// 如果有违规，添加到结果中
	if len(violations) > 0 {
		allPassed = false
		// 在结果前插入DAG验证结果
		dagResult := StepResult{
			Step:       "DAG Validation",
			Call:       "internal",
			Status:     "FAIL",
			Message:    fmt.Sprintf("Detected %d DAG constraint violations", len(violations)),
			Violations: violations,
		}
		results = append([]StepResult{dagResult}, results...)

	}

	return results, allPassed
}

// validateGraphAgainstTrace validates DAG format against trace data
func validateGraphAgainstTrace(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace, config spec.ValidationConfig, eval *evaluation) ([]StepResult, bool) {
	if err := fs.Graph.ValidateGraphStructure(); err != nil {
		return []StepResult{{Step: "graph-structure", Call: "internal", Status: "FAIL", Message: err.Error()}}, false
	}
	var results []StepResult
	okAll := true

	// Build call graph and validate DAG constraints first
	graph, err := BuildCallGraph(tr.Spans)
	if err != nil {
		return []StepResult{{
			Step:    "graph-build",
			Call:    "internal",
			Status:  "FAIL",
			Message: fmt.Sprintf("Failed to build call graph: %v", err),
		}}, false
	}

	// Validate DAG constraints (cycle detection, edge constraints)
	toleranceNanos := config.ToleranceMs * 1000000
	var violations []EdgeViolation
	if CausalityMode(config.Causality) != CausalityOff {
		violations = graph.ValidateEdgeConstraints(toleranceNanos)
	}
	if len(violations) > 0 {
		okAll = false
		// Add violations as a result
		dagResult := StepResult{
			Step:       "DAG Validation",
			Call:       "internal",
			Status:     "FAIL",
			Message:    fmt.Sprintf("Detected %d DAG constraint violations", len(violations)),
			Violations: violations,
		}
		results = append(results, dagResult)

	}

	// Build span matching index by service.operation
	spanIndex := make(map[string][]trace.Span)
	for _, span := range tr.Spans {
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
			results = append(results, StepResult{
				Step:    node.ID,
				Call:    node.Call,
				Status:  "FAIL",
				Message: fmt.Sprintf("DAG topological sort failed: %v", err),
			})
		}
		return results, false
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
	exports := map[string]map[string]any{}
	// Track matched spans to avoid double-matching
	usedSpans := make(map[string]bool) // span service:name:startNanos

	for _, nodeID := range topOrder {
		if eval.ctx.Err() != nil {
			return results, false
		}
		node := findNodeByID(fs.Graph, nodeID)
		if node == nil {
			results = append(results, StepResult{
				Step:    nodeID,
				Call:    "",
				Status:  "FAIL",
				Message: "Node not found",
			})
			okAll = false
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
			results = append(results, StepResult{Step: node.ID, Call: node.Call, Status: "FAIL", Message: message})
			okAll = false
			continue
		}

		usedSpans[getSpanID(*matchedSpan)] = true
		matchedSpans[node.ID] = matchedSpan

		step := spec.FlowStep{Step: node.ID, Call: node.Call, Input: node.Input, Output: node.Output, Meta: node.Meta}
		result := StepResult{Step: node.ID, Call: node.Call, Status: "PASS"}
		vars, err := graphVariables(fs.Graph, node.ID, topOrder, exports)
		if err != nil {
			result.Status, result.Message = "FAIL", err.Error()
		} else {
			result, exports[node.ID] = evaluateStep(result, step, *matchedSpan, opIndex, vars, config, eval)
		}
		okAll = okAll && result.Status == "PASS"
		results = append(results, result)
	}

	return results, okAll
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
