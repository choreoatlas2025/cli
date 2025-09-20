package validate

import (
	"fmt"
	"sort"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

// CallGraph 表示调用关系图
type CallGraph struct {
	Nodes map[string]*CallNode `json:"nodes"`
	Edges []*CallEdge          `json:"edges"`
}

// CallNode 表示调用图中的节点
type CallNode struct {
	SpanID     string            `json:"spanId"`
	TraceID    string            `json:"traceId"`
	Service    string            `json:"service"`
	Operation  string            `json:"operation"`
	StartNanos int64             `json:"startNanos"`
	EndNanos   int64             `json:"endNanos"`
	Attributes map[string]any    `json:"attributes"`
	Children   []*CallNode       `json:"children,omitempty"`
	Parent     *CallNode         `json:"parent,omitempty"`
}

// CallEdge 表示调用关系边
type CallEdge struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Relationship string `json:"relationship"` // "parent", "follows", "concurrent"
}

// ParallelStep 表示并发步骤
type ParallelStep struct {
	Parallel []spec.FlowStep `yaml:"parallel"`
}

// BuildCallGraph 从spans构建调用关系图
func BuildCallGraph(spans []trace.Span) (*CallGraph, error) {
	graph := &CallGraph{
		Nodes: make(map[string]*CallNode),
		Edges: make([]*CallEdge, 0),
	}

	// 创建节点
	for _, span := range spans {
		spanID := getSpanID(span)
		node := &CallNode{
			SpanID:     spanID,
			TraceID:    getTraceID(span),
			Service:    span.Service,
			Operation:  span.Name,
			StartNanos: span.StartNanos,
			EndNanos:   span.EndNanos,
			Attributes: span.Attributes,
			Children:   make([]*CallNode, 0),
		}
		graph.Nodes[spanID] = node
	}

	// 建立父子关系
	for _, span := range spans {
		spanID := getSpanID(span)
		parentSpanID := getParentSpanID(span)
		
		if parentSpanID != "" && parentSpanID != spanID {
			if parentNode, exists := graph.Nodes[parentSpanID]; exists {
				if childNode, exists := graph.Nodes[spanID]; exists {
					childNode.Parent = parentNode
					parentNode.Children = append(parentNode.Children, childNode)
					
					graph.Edges = append(graph.Edges, &CallEdge{
						From:         parentSpanID,
						To:          spanID,
						Relationship: "parent",
					})
				}
			}
		}
	}

	// 建立时序关系（同级spans的先后顺序）
	buildTemporalEdges(graph)

	return graph, nil
}

// buildTemporalEdges 建立时序边
func buildTemporalEdges(graph *CallGraph) {
	// 按开始时间排序所有节点
	var allNodes []*CallNode
	for _, node := range graph.Nodes {
		allNodes = append(allNodes, node)
	}
	sort.Slice(allNodes, func(i, j int) bool {
		return allNodes[i].StartNanos < allNodes[j].StartNanos
	})

	// 为同级节点建立时序关系
	for i := 0; i < len(allNodes)-1; i++ {
		current := allNodes[i]
		next := allNodes[i+1]
		
		// 如果两个节点有相同的父节点或都是顶级节点
		if hasSameParent(current, next) {
			if current.EndNanos <= next.StartNanos {
				// 顺序执行
				graph.Edges = append(graph.Edges, &CallEdge{
					From:         current.SpanID,
					To:          next.SpanID,
					Relationship: "follows",
				})
			} else if isOverlapping(current, next) {
				// 并发执行
				graph.Edges = append(graph.Edges, &CallEdge{
					From:         current.SpanID,
					To:          next.SpanID,
					Relationship: "concurrent",
				})
			}
		}
	}
}

// CheckCausality 检查因果关系和并发约束
func CheckCausality(flow *spec.FlowSpec, graph *CallGraph) ([]StepResult, bool) {
	var results []StepResult
	allPassed := true

	// 处理常规流程步骤
	for _, step := range flow.Flow {
		if len(step.Parallel) > 0 {
			// 并发步骤组 - 优先处理并发步骤
			parallelResults := checkParallelSteps(step.Parallel, graph)
			results = append(results, parallelResults...)
			for _, pr := range parallelResults {
				if pr.Status != "PASS" {
					allPassed = false
				}
			}
		} else if step.Step != "" && step.Call != "" {
			// 常规步骤
			result := checkSingleStep(step, graph)
			results = append(results, result)
			if result.Status != "PASS" {
				allPassed = false
			}
		}
	}

	return results, allPassed
}

// checkSingleStep 检查单个步骤
func checkSingleStep(step spec.FlowStep, graph *CallGraph) StepResult {
	svc, op, err := splitCall(step.Call)
	if err != nil {
		return StepResult{
			Step:    step.Step,
			Call:    step.Call,
			Status:  "FAIL",
			Message: fmt.Sprintf("Failed to parse call: %v", err),
		}
	}

	// 在图中查找匹配的节点
	var matchedNode *CallNode
	for _, node := range graph.Nodes {
		if normalize(node.Service) == normalize(svc) && normalize(node.Operation) == normalize(op) {
			matchedNode = node
			break
		}
	}

	if matchedNode == nil {
		return StepResult{
			Step:    step.Step,
			Call:    step.Call,
			Status:  "FAIL",
			Message: "No matching span found in trace",
		}
	}

	return StepResult{
		Step:   step.Step,
		Call:   step.Call,
		Status: "PASS",
	}
}

// checkParallelSteps 检查并发步骤组
func checkParallelSteps(parallelSteps []spec.FlowStep, graph *CallGraph) []StepResult {
	var results []StepResult
	var matchedNodes []*CallNode

	// 找到所有并发步骤对应的节点
	for _, step := range parallelSteps {
		svc, op, err := splitCall(step.Call)
		if err != nil {
			results = append(results, StepResult{
				Step:    step.Step,
				Call:    step.Call,
				Status:  "FAIL",
				Message: fmt.Sprintf("Failed to parse call: %v", err),
			})
			continue
		}

		var matchedNode *CallNode
		for _, node := range graph.Nodes {
			if normalize(node.Service) == normalize(svc) && normalize(node.Operation) == normalize(op) {
				matchedNode = node
				break
			}
		}

		if matchedNode == nil {
			results = append(results, StepResult{
				Step:    step.Step,
				Call:    step.Call,
				Status:  "FAIL",
				Message: "No matching span found in trace",
			})
		} else {
			matchedNodes = append(matchedNodes, matchedNode)
			results = append(results, StepResult{
				Step:   step.Step,
				Call:   step.Call,
				Status: "PASS",
			})
		}
	}

	// 验证并发约束：所有步骤应该在时间上重叠或属于同一父span
	if len(matchedNodes) > 1 {
		if !validateConcurrency(matchedNodes) {
			// 更新所有相关结果为失败
			for i := range results {
				if results[i].Status == "PASS" {
					results[i].Status = "FAIL"
					results[i].Message = "Concurrency constraint violation: steps not executed concurrently"
				}
			}
		}
	}

	return results
}

// validateConcurrency 验证节点是否满足并发约束
func validateConcurrency(nodes []*CallNode) bool {
	if len(nodes) <= 1 {
		return true
	}

	// 检查是否有时间重叠
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if !isOverlapping(nodes[i], nodes[j]) && !hasSameParent(nodes[i], nodes[j]) {
				return false
			}
		}
	}

	return true
}

// 辅助函数
func getSpanID(span trace.Span) string {
	if spanID, exists := span.Attributes["otlp.span_id"]; exists {
		if str, ok := spanID.(string); ok {
			return str
		}
	}
	// 生成唯一ID：service:operation:timestamp
	return fmt.Sprintf("%s:%s:%d", span.Service, span.Name, span.StartNanos)
}

func getTraceID(span trace.Span) string {
	if traceID, exists := span.Attributes["otlp.trace_id"]; exists {
		if str, ok := traceID.(string); ok {
			return str
		}
	}
	return "unknown-trace"
}

func getParentSpanID(span trace.Span) string {
	if parentSpanID, exists := span.Attributes["otlp.parent_span_id"]; exists {
		if str, ok := parentSpanID.(string); ok {
			return str
		}
	}
	return ""
}

func hasSameParent(node1, node2 *CallNode) bool {
	if node1.Parent == nil && node2.Parent == nil {
		return true // 都是根节点
	}
	if node1.Parent != nil && node2.Parent != nil {
		return node1.Parent.SpanID == node2.Parent.SpanID
	}
	return false
}

func isOverlapping(node1, node2 *CallNode) bool {
	// 检查时间区间是否重叠
	return node1.StartNanos < node2.EndNanos && node2.StartNanos < node1.EndNanos
}

// ValidateSequentialSteps 验证顺序步骤（增强版的原有逻辑）
func ValidateSequentialSteps(flow *spec.FlowSpec, spans []trace.Span) ([]StepResult, bool) {
	// 构建调用图
	graph, err := BuildCallGraph(spans)
	if err != nil {
		return []StepResult{{
			Step:    "graph-build",
			Call:    "internal",
			Status:  "FAIL",
			Message: fmt.Sprintf("Failed to build call graph: %v", err),
		}}, false
	}

	// 使用新的因果检查逻辑
	return CheckCausality(flow, graph)
}

// GetCallGraphStats 获取调用图统计信息
func GetCallGraphStats(graph *CallGraph) map[string]any {
	stats := map[string]any{
		"totalNodes":      len(graph.Nodes),
		"totalEdges":      len(graph.Edges),
		"rootNodes":       0,
		"maxDepth":        0,
		"concurrentPairs": 0,
		"services":        make(map[string]int),
	}

	serviceStats := make(map[string]int)
	rootCount := 0
	concurrentCount := 0

	for _, node := range graph.Nodes {
		serviceStats[node.Service]++
		if node.Parent == nil {
			rootCount++
		}
	}

	for _, edge := range graph.Edges {
		if edge.Relationship == "concurrent" {
			concurrentCount++
		}
	}

	stats["rootNodes"] = rootCount
	stats["concurrentPairs"] = concurrentCount
	stats["services"] = serviceStats

	return stats
}

// EdgeViolation 表示边约束违规
type EdgeViolation struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Type    string `json:"type"`    // "cycle", "causality", "overlap"
	Message string `json:"message"`
}

// DetectCycle 检测调用图中的循环依赖
func (g *CallGraph) DetectCycle() (bool, []string) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cyclePath []string

	// 对每个节点进行DFS
	for spanID := range g.Nodes {
		if !visited[spanID] {
			if path := g.dfsDetectCycle(spanID, visited, recStack, []string{}); path != nil {
				cyclePath = path
				return true, cyclePath
			}
		}
	}

	return false, nil
}

// dfsDetectCycle DFS辅助函数检测循环
func (g *CallGraph) dfsDetectCycle(spanID string, visited, recStack map[string]bool, path []string) []string {
	visited[spanID] = true
	recStack[spanID] = true
	path = append(path, spanID)

	// 检查所有边
	for _, edge := range g.Edges {
		if edge.From == spanID && edge.Relationship != "concurrent" {
			if !visited[edge.To] {
				if cyclePath := g.dfsDetectCycle(edge.To, visited, recStack, path); cyclePath != nil {
					return cyclePath
				}
			} else if recStack[edge.To] {
				// 找到循环，构建循环路径
				cycleStart := -1
				for i, id := range path {
					if id == edge.To {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					return append(path[cycleStart:], edge.To)
				}
			}
		}
	}

	recStack[spanID] = false
	return nil
}

// ValidateEdgeConstraints 验证边约束（包括容差）
func (g *CallGraph) ValidateEdgeConstraints(toleranceNanos int64) []EdgeViolation {
	var violations []EdgeViolation

	// 1. 检测循环
	if hasCycle, cyclePath := g.DetectCycle(); hasCycle {
		violations = append(violations, EdgeViolation{
			From:    cyclePath[len(cyclePath)-2],
			To:      cyclePath[len(cyclePath)-1],
			Type:    "cycle",
			Message: fmt.Sprintf("Cycle detected: %v", cyclePath),
		})
	}

	// 2. 验证因果约束
	for _, edge := range g.Edges {
		fromNode := g.Nodes[edge.From]
		toNode := g.Nodes[edge.To]

		if fromNode == nil || toNode == nil {
			continue
		}

		switch edge.Relationship {
		case "follows":
			// 验证时序关系：from应该在to之前结束（考虑容差）
			if fromNode.EndNanos > toNode.StartNanos+toleranceNanos {
				violations = append(violations, EdgeViolation{
					From: edge.From,
					To:   edge.To,
					Type: "causality",
					Message: fmt.Sprintf("Causality constraint violation: %s.%s should complete before %s.%s (tolerance %dms)",
						fromNode.Service, fromNode.Operation,
						toNode.Service, toNode.Operation,
						toleranceNanos/1000000),
				})
			}
		case "parent":
			// 验证父子关系：子节点应在父节点时间范围内
			if toNode.StartNanos < fromNode.StartNanos-toleranceNanos ||
				toNode.EndNanos > fromNode.EndNanos+toleranceNanos {
				violations = append(violations, EdgeViolation{
					From: edge.From,
					To:   edge.To,
					Type: "parent-child",
					Message: fmt.Sprintf("Parent-child constraint violation: %s.%s should be within parent %s.%s time range",
						toNode.Service, toNode.Operation,
						fromNode.Service, fromNode.Operation),
				})
			}
		case "concurrent":
			// 验证并发关系：应有时间重叠
			if !isOverlapping(fromNode, toNode) {
				violations = append(violations, EdgeViolation{
					From: edge.From,
					To:   edge.To,
					Type: "overlap",
					Message: fmt.Sprintf("Concurrency constraint violation: %s.%s and %s.%s should overlap but don't",
						fromNode.Service, fromNode.Operation,
						toNode.Service, toNode.Operation),
				})
			}
		}
	}

	return violations
}

// GetTopologicalOrder 获取拓扑排序
func (g *CallGraph) GetTopologicalOrder() ([]string, error) {
	// 计算入度
	inDegree := make(map[string]int)
	for spanID := range g.Nodes {
		inDegree[spanID] = 0
	}

	for _, edge := range g.Edges {
		if edge.Relationship != "concurrent" {
			inDegree[edge.To]++
		}
	}

	// 找出所有入度为0的节点
	var queue []string
	for spanID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, spanID)
		}
	}

	var sorted []string
	processedCount := 0

	// 执行拓扑排序
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)
		processedCount++

		// 更新相邻节点的入度
		for _, edge := range g.Edges {
			if edge.From == current && edge.Relationship != "concurrent" {
				inDegree[edge.To]--
				if inDegree[edge.To] == 0 {
					queue = append(queue, edge.To)
				}
			}
		}
	}

	if processedCount != len(g.Nodes) {
		return nil, fmt.Errorf("Cannot complete topological sort: cycle detected in graph")
	}

	return sorted, nil
}