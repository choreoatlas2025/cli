// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConvertGraphToFlow accepts only an unconditional chain. A DAG's independent
// branches need not overlap, while Flow.parallel requires overlap; converting
// branching graphs would change the accepted executions. Keep those in DAG form.
func ConvertGraphToFlow(fs *FlowSpec) (*FlowSpec, error) {
	if fs == nil || fs.Graph == nil {
		return nil, fmt.Errorf("invalid conversion input: a graph is required")
	}
	g := fs.Graph
	if len(g.Nodes) == 0 {
		return nil, fmt.Errorf("invalid graph: no nodes")
	}
	if err := g.ValidateGraphStructure(); err != nil {
		return nil, fmt.Errorf("invalid graph: %w", err)
	}
	inDegree := map[string]int{}
	successor := map[string]string{}
	nodes := map[string]GraphNode{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}
	for _, e := range g.Edges {
		if e.Condition != "" {
			return nil, fmt.Errorf("invalid conversion: conditional edges cannot be represented by Flow; retain DAG format")
		}
		inDegree[e.To]++
		if _, exists := successor[e.From]; exists || inDegree[e.To] > 1 {
			return nil, fmt.Errorf("invalid conversion: branching or joining cannot be represented equivalently by Flow; retain DAG format")
		}
		successor[e.From] = e.To
	}
	root := ""
	for _, n := range g.Nodes {
		if inDegree[n.ID] == 0 {
			if root != "" {
				return nil, fmt.Errorf("invalid conversion: independent roots cannot be represented equivalently by Flow; retain DAG format")
			}
			root = n.ID
		}
	}
	out := &FlowSpec{Info: fs.Info, Services: make(map[string]ServiceBinding, len(fs.Services))}
	for alias, binding := range fs.Services {
		out.Services[alias] = binding
	}
	for id := root; id != ""; id = successor[id] {
		out.Flow = append(out.Flow, nodeToStep(nodes[id]))
	}
	if len(out.Flow) != len(g.Nodes) {
		return nil, fmt.Errorf("invalid conversion: not all graph nodes were preserved")
	}
	return out, nil
}

func nodeToStep(n GraphNode) FlowStep {
	return FlowStep{Step: n.ID, Call: n.Call, Input: n.Input, Output: n.Output, Meta: n.Meta}
}

func WriteFlowSpec(path string, fs *FlowSpec) error {
	b, err := yaml.Marshal(fs)
	if err != nil {
		return fmt.Errorf("failed to marshal flowspec: %w", err)
	}
	return os.WriteFile(path, b, 0o644)
}
