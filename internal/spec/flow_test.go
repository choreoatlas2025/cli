// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
	"testing"
)

func TestGraphStructureValidation(t *testing.T) {
	tests := []struct {
		name      string
		graph     *GraphSpec
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid DAG",
			graph: &GraphSpec{
				Nodes: []GraphNode{
					{ID: "a", Call: "service.op1"},
					{ID: "b", Call: "service.op2"},
					{ID: "c", Call: "service.op3"},
				},
				Edges: []GraphEdge{
					{From: "a", To: "b"},
					{From: "b", To: "c"},
				},
			},
			expectErr: false,
		},
		{
			name: "Duplicate node IDs",
			graph: &GraphSpec{
				Nodes: []GraphNode{
					{ID: "a", Call: "service.op1"},
					{ID: "a", Call: "service.op2"}, // Duplicate ID
				},
				Edges: []GraphEdge{},
			},
			expectErr: true,
			errMsg:    "duplicate node ID: a",
		},
		{
			name: "Empty node ID",
			graph: &GraphSpec{
				Nodes: []GraphNode{
					{ID: "", Call: "service.op1"}, // Empty ID
				},
				Edges: []GraphEdge{},
			},
			expectErr: true,
			errMsg:    "node ID cannot be empty",
		},
		{
			name: "Edge references non-existent node",
			graph: &GraphSpec{
				Nodes: []GraphNode{
					{ID: "a", Call: "service.op1"},
				},
				Edges: []GraphEdge{
					{From: "a", To: "nonexistent"}, // Non-existent node
				},
			},
			expectErr: true,
			errMsg:    "edge references non-existent node: nonexistent",
		},
		{
			name: "Cycle in graph",
			graph: &GraphSpec{
				Nodes: []GraphNode{
					{ID: "a", Call: "service.op1"},
					{ID: "b", Call: "service.op2"},
					{ID: "c", Call: "service.op3"},
				},
				Edges: []GraphEdge{
					{From: "a", To: "b"},
					{From: "b", To: "c"},
					{From: "c", To: "a"}, // Creates cycle
				},
			},
			expectErr: true,
			errMsg:    "cycle detected in graph",
		},
		{
			name: "No entry nodes (all nodes have incoming edges)",
			graph: &GraphSpec{
				Nodes: []GraphNode{
					{ID: "a", Call: "service.op1"},
					{ID: "b", Call: "service.op2"},
					{ID: "c", Call: "service.op3"},
				},
				Edges: []GraphEdge{
					{From: "a", To: "b"}, // Linear: a->b->c, but we'll add an edge to make 'a' not an entry
					{From: "b", To: "c"},
					{From: "c", To: "a"}, // This makes 'a' not an entry node, creating a cycle
				},
			},
			expectErr: true,
			errMsg:    "cycle detected in graph", // Cycle will be detected first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.graph.ValidateGraphStructure()
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestFlowSpecGraphMode(t *testing.T) {
	// Test IsGraphMode function
	flowSpec := &FlowSpec{
		Flow: []FlowStep{{Step: "step1", Call: "service.op"}},
	}
	if flowSpec.IsGraphMode() {
		t.Errorf("Expected IsGraphMode() to return false for flow format")
	}

	graphSpec := &FlowSpec{
		Graph: &GraphSpec{
			Nodes: []GraphNode{{ID: "node1", Call: "service.op"}},
		},
	}
	if !graphSpec.IsGraphMode() {
		t.Errorf("Expected IsGraphMode() to return true for graph format")
	}
}

func TestGetStepsCount(t *testing.T) {
	flowSpec := &FlowSpec{
		Flow: []FlowStep{
			{Step: "step1", Call: "service.op1"},
			{Step: "step2", Call: "service.op2"},
		},
	}
	if count := flowSpec.GetStepsCount(); count != 2 {
		t.Errorf("Expected 2 steps, got %d", count)
	}

	graphSpec := &FlowSpec{
		Graph: &GraphSpec{
			Nodes: []GraphNode{
				{ID: "node1", Call: "service.op1"},
				{ID: "node2", Call: "service.op2"},
				{ID: "node3", Call: "service.op3"},
			},
		},
	}
	if count := graphSpec.GetStepsCount(); count != 3 {
		t.Errorf("Expected 3 nodes, got %d", count)
	}
}

func TestGetStepNames(t *testing.T) {
	flowSpec := &FlowSpec{
		Flow: []FlowStep{
			{Step: "step1", Call: "service.op1"},
			{Step: "step2", Call: "service.op2"},
		},
	}
	names := flowSpec.GetStepNames()
	expectedNames := []string{"step1", "step2"}
	
	if len(names) != len(expectedNames) {
		t.Errorf("Expected %d names, got %d", len(expectedNames), len(names))
	}
	
	for i, name := range names {
		if name != expectedNames[i] {
			t.Errorf("Expected name '%s' at index %d, got '%s'", expectedNames[i], i, name)
		}
	}

	graphSpec := &FlowSpec{
		Graph: &GraphSpec{
			Nodes: []GraphNode{
				{ID: "nodeA", Call: "service.op1"},
				{ID: "nodeB", Call: "service.op2"},
			},
		},
	}
	names = graphSpec.GetStepNames()
	expectedNames = []string{"nodeA", "nodeB"}
	
	if len(names) != len(expectedNames) {
		t.Errorf("Expected %d names, got %d", len(expectedNames), len(names))
	}
	
	for i, name := range names {
		if name != expectedNames[i] {
			t.Errorf("Expected name '%s' at index %d, got '%s'", expectedNames[i], i, name)
		}
	}
}