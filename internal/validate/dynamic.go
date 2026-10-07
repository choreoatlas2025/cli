// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"fmt"
	"github.com/choreoatlas2025/cli/internal/verdict"

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
		return nil, []StepResult{{Step: "contract-plan", Call: "internal", Status: "FAIL", Issue: verdict.InvalidContract, Message: err.Error()}}, false
	}
	results, passed := plan.ValidateTrace(tr)
	return plan, results, passed
}

// validateAgainstTrace coordinates structural matching and shared evaluation.
// Neither matcher knows CEL, contract operations or exported values.
func validateAgainstTrace(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace, config spec.ValidationConfig, eval *evaluation) ([]StepResult, bool) {
	if fs.Graph != nil {
		flowCopy, graphCopy := *fs, *fs.Graph
		graphCopy.Edges = append([]spec.GraphEdge(nil), fs.Graph.Edges...)
		flowCopy.Graph = &graphCopy
		fs = &flowCopy
	}
	if err := trace.ValidateTimestamps(tr.Spans, CausalityMode(config.Causality) != CausalityOff || hasParallel(fs.Flow) || hasConcurrentEdges(fs.Graph)); err != nil {
		issue := verdict.StructureMismatch
		for _, span := range tr.Spans {
			if !span.HasStart() || !span.HasEnd() {
				issue = verdict.MissingEvidence
				break
			}
		}
		return []StepResult{{Step: "trace-time", Call: "internal", Status: "FAIL", Issue: issue, Message: err.Error()}}, false
	}
	calls, err := BuildCallGraph(tr.Spans)
	if err != nil {
		return []StepResult{{Step: "graph-build", Call: "internal", Status: "FAIL", Issue: verdict.StructureMismatch, Message: fmt.Sprintf("Failed to build call graph: %v", err)}}, false
	}
	var results []StepResult
	if config.Causality != "off" {
		if violations := calls.ValidateEdgeConstraints(config.ToleranceMs * 1000000); len(violations) > 0 {
			results = append(results, StepResult{Step: "DAG Validation", Call: "internal", Status: "FAIL", Issue: verdict.StructureMismatch, Message: fmt.Sprintf("Detected %d DAG constraint violations", len(violations)), Violations: violations})
		}
	}
	if fs.IsGraphMode() {
		results = append(results, evaluateGraphMatches(matchGraphSteps(fs, calls, config, eval.ctx), fs.Graph, opIndex, config, eval)...)
	} else {
		results = append(results, evaluateFlowMatches(matchFlowSteps(fs, calls, config), opIndex, config, eval)...)
	}
	return results, AllStepsPassed(results)
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
