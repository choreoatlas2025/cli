// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FlowSpec represents flow specification
type FlowSpec struct {
	Info     FlowInfo                  `yaml:"info"`
	Services map[string]ServiceBinding `yaml:"services"`
	Flow     []FlowStep                `yaml:"flow,omitempty"`  // Legacy flow format
	Graph    *GraphSpec                `yaml:"graph,omitempty"` // New DAG format
}

// FlowInfo contains basic flow information
type FlowInfo struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version,omitempty"`
}

// ServiceBinding represents service binding configuration
type ServiceBinding struct {
	Spec string `yaml:"spec"` // Path to service.spec.yaml
}

// FlowStep represents a step in the flow
type FlowStep struct {
	Step     string                 `yaml:"step,omitempty"`
	Call     string                 `yaml:"call,omitempty"`     // Format: "userService.createUser"
	Input    map[string]any         `yaml:"input,omitempty"`    // Supports ${var} references
	Output   map[string]string      `yaml:"output,omitempty"`   // Output mappings e.g. { newUserResponse: "response.body" }
	Meta     map[string]interface{} `yaml:"meta,omitempty"`     // Reserved for metadata
	Parallel []FlowStep             `yaml:"parallel,omitempty"` // Parallel step group
}

// GraphSpec represents DAG format flow specification
type GraphSpec struct {
	Nodes   []GraphNode `yaml:"nodes"`
	Edges   []GraphEdge `yaml:"edges,omitempty"` // Optional explicit edges
	ensured bool        `yaml:"-"`               // Internal flag to track if edges are built
}

// GraphNode represents a node in the DAG
type GraphNode struct {
	ID      string                 `yaml:"id"`
	Call    string                 `yaml:"call"`
	Depends []string               `yaml:"depends,omitempty"` // Node IDs this node depends on
	Input   map[string]any         `yaml:"input,omitempty"`
	Output  map[string]string      `yaml:"output,omitempty"`
	Meta    map[string]interface{} `yaml:"meta,omitempty"`
}

// GraphEdge represents an edge in the DAG
type GraphEdge struct {
	From      string `yaml:"from"`
	To        string `yaml:"to"`
	Condition string `yaml:"condition,omitempty"` // Nonempty conditions are unsupported in CE.
	// An explicit relationship retains its meaning in either strict or temporal
	// mode. Empty relationships retain the existing --causality interpretation.
	Relationship string `yaml:"relationship,omitempty"`
}

// CarriesOutputs distinguishes completion dependencies from relationships
// which only constrain span structure. A parent's response is not available to
// its child, nor is a concurrent call's response available to its sibling.
func (e GraphEdge) CarriesOutputs() bool {
	return e.Relationship == "" || e.Relationship == "follows"
}

// LoadFlowSpec loads flow specification from file
func LoadFlowSpec(path string) (*FlowSpec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read flowspec: %w", err)
	}
	return ParseFlowSpec(b)
}

func ParseFlowSpec(b []byte) (*FlowSpec, error) {
	// Try to parse with graph format first (preferred)
	var fs FlowSpec
	if err := yaml.Unmarshal(b, &fs); err != nil {
		return nil, fmt.Errorf("failed to parse flowspec: %w", err)
	}

	// Validate that either graph or flow is present (not both)
	if fs.Graph != nil && len(fs.Flow) > 0 {
		return nil, fmt.Errorf("flowspec cannot have both 'graph' and 'flow' fields")
	}

	if fs.Graph == nil && len(fs.Flow) == 0 {
		return nil, fmt.Errorf("flowspec must have either 'graph' or 'flow' field")
	}

	// If using graph format, ensure edges are built
	if fs.Graph != nil {
		fs.Graph.EnsureEdges()
		if err := fs.Graph.validateEdgeConditions(); err != nil {
			return nil, err
		}
	}

	return &fs, nil
}

// BuildOperationIndex builds service operation index
func (fs *FlowSpec) BuildOperationIndex(flowPath string) (map[string]*ServiceSpecFile, map[string]map[string]ServiceOperation, error) {
	base := filepath.Dir(flowPath)
	serviceFiles := make(map[string]*ServiceSpecFile)
	opIndex := make(map[string]map[string]ServiceOperation)

	for alias, bind := range fs.Services {
		specPath := bind.Spec
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(base, specPath)
		}
		ss, err := LoadServiceSpec(specPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load service '%s' spec: %w", alias, err)
		}
		serviceFiles[alias] = ss
		ops := make(map[string]ServiceOperation)
		for _, op := range ss.Operations {
			ops[op.OperationId] = op
		}
		opIndex[alias] = ops
	}
	return serviceFiles, opIndex, nil
}

// IsGraphMode returns true if this flowspec uses the new graph format
func (fs *FlowSpec) IsGraphMode() bool {
	return fs.Graph != nil
}

// GetStepsCount returns the total number of steps/nodes in the flowspec
func (fs *FlowSpec) GetStepsCount() int {
	return len(fs.CallSteps())
}

// GetStepNames returns all step names in the flowspec
func (fs *FlowSpec) GetStepNames() []string {
	steps := fs.CallSteps()
	names := make([]string, len(steps))
	for i, step := range steps {
		names[i] = step.Step
	}
	return names
}

// CallSteps counts actual calls, not containers. A parent with a call and
// parallel children contributes both its own call and every child call.
func (fs *FlowSpec) CallSteps() []FlowStep {
	var steps []FlowStep
	if fs.IsGraphMode() {
		for _, node := range fs.Graph.Nodes {
			steps = append(steps, nodeToStep(node))
		}
		return steps
	}
	var collect func([]FlowStep)
	collect = func(flow []FlowStep) {
		for _, step := range flow {
			if step.Call != "" {
				steps = append(steps, step)
			}
			collect(step.Parallel)
		}
	}
	collect(fs.Flow)
	return steps
}

// ValidateGraphStructure validates the DAG structure
func (gs *GraphSpec) ValidateGraphStructure() error {
	if gs == nil {
		return fmt.Errorf("graph is nil")
	}

	// Ensure edges are built from depends field
	gs.EnsureEdges()
	if err := gs.validateEdgeConditions(); err != nil {
		return err
	}

	// Build node ID set
	nodeIDs := make(map[string]bool)
	for _, node := range gs.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true
	}

	// Validate edges reference existing nodes
	for _, edge := range gs.Edges {
		if !nodeIDs[edge.From] {
			return fmt.Errorf("edge references non-existent node: %s", edge.From)
		}
		if !nodeIDs[edge.To] {
			return fmt.Errorf("edge references non-existent node: %s", edge.To)
		}
	}

	// Check for cycles using DFS
	if err := gs.checkCycles(); err != nil {
		return err
	}

	// Check connectivity (all nodes reachable from entry nodes)
	if err := gs.checkConnectivity(); err != nil {
		return err
	}

	return nil
}

func (gs *GraphSpec) validateEdgeConditions() error {
	for _, edge := range gs.Edges {
		switch edge.Relationship {
		case "", "parent", "follows", "concurrent":
		default:
			return fmt.Errorf("invalid contract: unsupported edge relationship %q", edge.Relationship)
		}
		if edge.Condition != "" {
			return fmt.Errorf("invalid contract: unsupported conditional edge %s -> %s: CE supports unconditional dependencies only", edge.From, edge.To)
		}
	}
	return nil
}

// EnsureEdges normalizes the union of explicit edges and node.depends.
// Identical edges are deduplicated; conditional edges retain their conditions,
// and a depends entry always contributes an unconditional dependency.
func (gs *GraphSpec) EnsureEdges() {
	if gs == nil || gs.ensured {
		return
	}
	edges := make([]GraphEdge, 0, len(gs.Edges))
	seen := map[GraphEdge]bool{}
	add := func(edge GraphEdge) {
		if !seen[edge] {
			seen[edge] = true
			edges = append(edges, edge)
		}
	}
	for _, edge := range gs.Edges {
		add(edge)
	}
	for _, node := range gs.Nodes {
		for _, predecessor := range node.Depends {
			add(GraphEdge{From: predecessor, To: node.ID})
		}
	}
	gs.Edges = edges
	gs.ensured = true
}

// checkCycles uses DFS to detect cycles in the graph
func (gs *GraphSpec) checkCycles() error {
	// Build adjacency list
	adj := make(map[string][]string)
	for _, edge := range gs.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
	}

	// DFS cycle detection
	white := make(map[string]bool) // unvisited
	gray := make(map[string]bool)  // visiting
	black := make(map[string]bool) // visited

	for _, node := range gs.Nodes {
		white[node.ID] = true
	}

	var dfs func(string) error
	dfs = func(nodeID string) error {
		if black[nodeID] {
			return nil
		}
		if gray[nodeID] {
			return fmt.Errorf("cycle detected in graph")
		}

		gray[nodeID] = true
		delete(white, nodeID)

		for _, neighbor := range adj[nodeID] {
			if err := dfs(neighbor); err != nil {
				return err
			}
		}

		delete(gray, nodeID)
		black[nodeID] = true
		return nil
	}

	for nodeID := range white {
		if err := dfs(nodeID); err != nil {
			return err
		}
	}

	return nil
}

// checkConnectivity ensures all nodes are reachable from entry nodes
func (gs *GraphSpec) checkConnectivity() error {
	// Build adjacency list and in-degree map
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all nodes
	for _, node := range gs.Nodes {
		inDegree[node.ID] = 0
	}

	// Build adjacency list and calculate in-degrees
	for _, edge := range gs.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
		inDegree[edge.To]++
	}

	// Find entry nodes (in-degree = 0)
	var entryNodes []string
	for nodeID, degree := range inDegree {
		if degree == 0 {
			entryNodes = append(entryNodes, nodeID)
		}
	}

	if len(entryNodes) == 0 {
		return fmt.Errorf("DAG must have at least one entry node (in-degree = 0)")
	}

	// BFS to check reachability
	visited := make(map[string]bool)
	queue := append([]string{}, entryNodes...)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		for _, neighbor := range adj[current] {
			if !visited[neighbor] {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check if all nodes are reachable
	for _, node := range gs.Nodes {
		if !visited[node.ID] {
			return fmt.Errorf("node %s is not reachable from entry nodes", node.ID)
		}
	}

	return nil
}
