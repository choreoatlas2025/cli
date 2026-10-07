// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/verdict"
	"sort"
)

// matchedStep keeps structural matching and semantic evaluation bound to
// the same call instance. A matched span can be consumed only once.
type matchedStep struct {
	stage  int
	step   spec.FlowStep
	node   *CallNode
	result StepResult
}

type flowMatcher struct {
	nodes    []*CallNode
	used     map[*CallNode]bool
	previous []*CallNode
	config   spec.ValidationConfig
}

func newFlowMatcher(graph *CallGraph, config spec.ValidationConfig) *flowMatcher {
	m := &flowMatcher{used: map[*CallNode]bool{}, config: config}
	for _, node := range graph.Nodes {
		m.nodes = append(m.nodes, node)
	}
	sort.Slice(m.nodes, func(i, j int) bool {
		a, b := m.nodes[i], m.nodes[j]
		if a.StartNanos != b.StartNanos {
			return a.StartNanos < b.StartNanos
		}
		if a.EndNanos != b.EndNanos {
			return a.EndNanos < b.EndNanos
		}
		return a.SpanID < b.SpanID
	})
	return m
}

func (m *flowMatcher) match(step spec.FlowStep) matchedStep {
	r := matchedStep{step: step, result: StepResult{Step: step.Step, Call: step.Call, Status: "FAIL", Issue: verdict.StructureMismatch}}
	svc, op, err := splitCall(step.Call)
	if err != nil {
		r.result.Message = err.Error()
		return r
	}
	for _, node := range m.nodes {
		span := trace.Span{Service: node.Service, Name: node.Operation, Attributes: node.Attributes}
		if m.used[node] || normalize(node.Service) != normalize(svc) || !spec.OperationMatches(op, span) {
			continue
		}
		valid := true
		for _, pred := range m.previous {
			switch CausalityMode(m.config.Causality) {
			case CausalityStrict:
				valid = valid && node.Parent == pred
			case CausalityTemporal:
				valid = valid && completesBefore(pred.EndNanos, node.StartNanos, m.config.ToleranceMs)
			}
		}
		if !valid {
			continue
		}
		m.used[node] = true
		r.node, r.result.Status, r.result.Issue = node, "PASS", ""
		return r
	}
	r.result.Message = "no unused matching span satisfies step order and causality"
	return r
}

func (m *flowMatcher) parallel(steps []spec.FlowStep) []matchedStep {
	var matched []matchedStep
	var nodes []*CallNode
	for _, step := range steps {
		r := m.match(step)
		matched = append(matched, r)
		if r.node != nil {
			nodes = append(nodes, r.node)
		}
	}
	if !validateConcurrency(nodes) {
		for i := range matched {
			if matched[i].result.Status == "PASS" {
				matched[i].result.Status = "FAIL"
				matched[i].result.Issue = verdict.StructureMismatch
				matched[i].result.Message = "concurrency constraint violation: spans do not overlap"
			}
		}
	}
	if len(nodes) > 0 {
		m.previous = nodes
	}
	return matched
}

func matchFlowSteps(flow *spec.FlowSpec, graph *CallGraph, config spec.ValidationConfig) []matchedStep {
	m := newFlowMatcher(graph, config)
	var matched []matchedStep
	stage := 0
	for _, step := range flow.Flow {
		if step.Call != "" || len(step.Parallel) == 0 {
			r := m.match(step)
			r.stage = stage
			stage++
			matched = append(matched, r)
			if r.node != nil {
				m.previous = []*CallNode{r.node}
			}
		}
		if len(step.Parallel) > 0 {
			children := m.parallel(step.Parallel)
			for i := range children {
				children[i].stage = stage
			}
			stage++
			matched = append(matched, children...)
		}
	}
	return matched
}
