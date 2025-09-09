package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FlowSpec 表示流程规约
type FlowSpec struct {
	Info     FlowInfo                  `yaml:"info"`
	Services map[string]ServiceBinding `yaml:"services"`
	Flow     []FlowStep                `yaml:"flow,omitempty"`    // Legacy flow format
	Graph    *GraphSpec               `yaml:"graph,omitempty"`   // New DAG format
}

// FlowInfo 包含流程的基本信息
type FlowInfo struct {
	Title string `yaml:"title"`
}

// ServiceBinding 表示服务绑定配置
type ServiceBinding struct {
	Spec string `yaml:"spec"` // 指向 service.spec.yaml
}

// FlowStep 表示流程中的一个步骤
type FlowStep struct {
	Step     string                 `yaml:"step,omitempty"`
	Call     string                 `yaml:"call,omitempty"`           // 形如 "userService.createUser"
	Input    map[string]any         `yaml:"input,omitempty"`          // 支持 ${var} 引用
	Output   map[string]string      `yaml:"output,omitempty"`         // 令牌归档，如 { newUserResponse: "response.body" }
	Meta     map[string]interface{} `yaml:"meta,omitempty"`           // 预留
	Parallel []FlowStep             `yaml:"parallel,omitempty"`       // 并发步骤组
}

// GraphSpec 表示DAG格式的流程规约
type GraphSpec struct {
	Nodes []GraphNode `yaml:"nodes"`
	Edges []GraphEdge `yaml:"edges"`
}

// GraphNode 表示DAG中的一个节点
type GraphNode struct {
	ID     string                 `yaml:"id"`
	Call   string                 `yaml:"call"`
	Input  map[string]any         `yaml:"input,omitempty"`
	Output map[string]string      `yaml:"output,omitempty"`
	Meta   map[string]interface{} `yaml:"meta,omitempty"`
}

// GraphEdge 表示DAG中的一条边
type GraphEdge struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// LoadFlowSpec 从文件加载流程规约
func LoadFlowSpec(path string) (*FlowSpec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 flowspec 失败: %w", err)
	}
	var fs FlowSpec
	if err := yaml.Unmarshal(b, &fs); err != nil {
		return nil, fmt.Errorf("解析 flowspec 失败: %w", err)
	}
	return &fs, nil
}

// BuildOperationIndex 构建服务操作索引
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
			return nil, nil, fmt.Errorf("加载服务 '%s' 规约失败: %w", alias, err)
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
	if fs.IsGraphMode() {
		return len(fs.Graph.Nodes)
	}
	return len(fs.Flow)
}

// GetStepNames returns all step names in the flowspec
func (fs *FlowSpec) GetStepNames() []string {
	if fs.IsGraphMode() {
		names := make([]string, len(fs.Graph.Nodes))
		for i, node := range fs.Graph.Nodes {
			names[i] = node.ID
		}
		return names
	}
	
	names := make([]string, len(fs.Flow))
	for i, step := range fs.Flow {
		names[i] = step.Step
	}
	return names
}

// ValidateGraphStructure validates the DAG structure
func (gs *GraphSpec) ValidateGraphStructure() error {
	if gs == nil {
		return fmt.Errorf("graph is nil")
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
