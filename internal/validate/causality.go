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
			Message: fmt.Sprintf("解析调用失败: %v", err),
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
			Message: "未在 trace 中找到对应 span",
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
				Message: fmt.Sprintf("解析调用失败: %v", err),
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
				Message: "未在 trace 中找到对应 span",
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
					results[i].Message = "并发约束验证失败：步骤未并发执行"
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
			Message: fmt.Sprintf("构建调用图失败: %v", err),
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