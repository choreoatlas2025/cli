package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

// EnableSemantic 控制是否启用语义校验
var EnableSemantic = true

// StepResult 表示单个步骤的验证结果
type StepResult struct {
	Step       string            `json:"step"`
	Call       string            `json:"call"`
	Status     string            `json:"status"` // PASS / FAIL
	Message    string            `json:"message,omitempty"`
	Conditions []ConditionResult `json:"conditions,omitempty"`
}

// CausalityMode represents the causality checking mode
type CausalityMode string

const (
	CausalityStrict   CausalityMode = "strict"   // Use parent-child span relationships
	CausalityTemporal CausalityMode = "temporal" // Use temporal ordering
	CausalityOff      CausalityMode = "off"      // Disable causality checking
)

// Global causality mode setting
var GlobalCausalityMode = CausalityTemporal

// GlobalCausalityToleranceMs 因果约束容差（毫秒）
var GlobalCausalityToleranceMs int64 = 50

// ValidateAgainstTrace 根据追踪数据验证流程执行（支持因果和并发校验）
func ValidateAgainstTrace(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace) ([]StepResult, bool) {
	// Route to appropriate validation based on format
	if fs.IsGraphMode() {
		return validateGraphAgainstTrace(fs, opIndex, tr)
	}
	
	// Legacy flow validation
	// 检查是否包含并发步骤，决定使用哪种校验策略
	hasParallelSteps := false
	for _, step := range fs.Flow {
		if len(step.Parallel) > 0 {
			hasParallelSteps = true
			break
		}
	}

	// 如果有并发步骤或OTLP数据（包含父子关系），使用因果校验
	if hasParallelSteps || hasOTLPMetadata(tr) {
		return validateWithCausality(fs, opIndex, tr)
	}

	// 否则使用原来的时序校验
	return validateWithTimeSequence(fs, opIndex, tr)
}

// hasOTLPMetadata 检查trace是否包含OTLP元数据（parentSpanId等）
func hasOTLPMetadata(tr *trace.Trace) bool {
	for _, span := range tr.Spans {
		if _, exists := span.Attributes["otlp.parent_span_id"]; exists {
			return true
		}
	}
	return false
}

// validateWithCausality 使用因果校验（支持并发）
func validateWithCausality(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace) ([]StepResult, bool) {
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
	toleranceNanos := GlobalCausalityToleranceMs * 1000000 // 转换为纳秒
	violations := graph.ValidateEdgeConstraints(toleranceNanos)

	// 执行因果校验
	results, allPassed := CheckCausality(fs, graph)

	// 如果有违规，添加到结果中
	if len(violations) > 0 {
		allPassed = false
		// 在结果前插入DAG验证结果
		dagResult := StepResult{
			Step:    "DAG Validation",
			Call:    "internal",
			Status:  "FAIL",
			Message: fmt.Sprintf("Detected %d DAG constraint violations", len(violations)),
		}
		results = append([]StepResult{dagResult}, results...)

		// 输出详细的违规信息
		for _, v := range violations {
			fmt.Printf("[DAG Violation] %s: %s\n", v.Type, v.Message)
		}
	}

	// 应用语义校验
	if EnableSemantic {
		for i := range results {
			if results[i].Status == "PASS" {
				svc, op, err := splitCall(results[i].Call)
				if err == nil {
					// 找到对应的span
					for _, span := range tr.Spans {
						if normalize(span.Service) == normalize(svc) && normalize(span.Name) == normalize(op) {
							if ops, ok := opIndex[svc]; ok {
								if opSpec, ok := ops[op]; ok {
									conds, okSem := EvaluateConditions(
										spec.FlowStep{Step: results[i].Step, Call: results[i].Call},
										opSpec, span, map[string]any{})
									results[i].Conditions = conds
									if !okSem {
										results[i].Status = "FAIL"
										if results[i].Message != "" {
											results[i].Message += " | "
										}
										results[i].Message += "Semantic validation failed"
										allPassed = false
									}
								}
							}
							break
						}
					}
				}
			}
		}
	}

	return results, allPassed
}

// validateWithTimeSequence 使用原来的时序校验（向后兼容）
func validateWithTimeSequence(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace) ([]StepResult, bool) {
	var results []StepResult
	okAll := true

	// 严格时序校验：按 StartNanos 升序排序 spans
	sortedSpans := make([]trace.Span, len(tr.Spans))
	copy(sortedSpans, tr.Spans)
	sort.Slice(sortedSpans, func(i, j int) bool {
		return sortedSpans[i].StartNanos < sortedSpans[j].StartNanos
	})

	spanIndex := 0 // 当前匹配的 span 索引

	for _, st := range fs.Flow {
		// 跳过并发步骤组（由因果校验处理）
		if len(st.Parallel) > 0 {
			continue
		}

		svc, op, err := splitCall(st.Call)
		if err != nil {
			results = append(results, StepResult{Step: st.Step, Call: st.Call, Status: "FAIL", Message: err.Error()})
			okAll = false
			continue
		}

		// 在剩余的 spans 中查找匹配项（保持顺序）
		found := false
		matchedIndex := -1

		for j := spanIndex; j < len(sortedSpans); j++ {
			sp := sortedSpans[j]
			if normalize(sp.Service) == normalize(svc) && normalize(sp.Name) == normalize(op) {
				found = true
				matchedIndex = j
				break
			}
		}

		if found {
			if matchedIndex >= spanIndex {
				note := ""
				if matchedIndex > spanIndex {
					note = fmt.Sprintf("按时序匹配到第 %d 个 span（存在中间插入 span）", matchedIndex+1)
				}
				
				// 默认 PASS（顺序已通过）
				sr := StepResult{Step: st.Step, Call: st.Call, Status: "PASS", Message: note}

				// 语义校验（如果有对应的 operation 规约）
				if EnableSemantic {
					if ops, ok := opIndex[svc]; ok {
						if opSpec, ok := ops[op]; ok {
							conds, okSem := EvaluateConditions(st, opSpec, sortedSpans[matchedIndex], /*vars*/ map[string]any{})
							sr.Conditions = conds
							if !okSem {
								sr.Status = "FAIL"
								if sr.Message != "" {
									sr.Message += " | "
								}
								sr.Message += "语义校验未通过"
							}
						}
					}
				}
				
				results = append(results, sr)
				spanIndex = matchedIndex + 1
			} else {
				// span 出现在上一步之前（时序倒退）
				results = append(results, StepResult{
					Step:    st.Step,
					Call:    st.Call,
					Status:  "FAIL",
					Message: "时序倒退：匹配到的 span 早于上一步骤",
				})
				okAll = false
			}
		} else {
			results = append(results, StepResult{Step: st.Step, Call: st.Call, Status: "FAIL", Message: "未在 trace 中匹配到对应 span"})
			okAll = false
		}
	}
	return results, okAll
}

// validateGraphAgainstTrace validates DAG format against trace data
func validateGraphAgainstTrace(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation, tr *trace.Trace) ([]StepResult, bool) {
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
	toleranceNanos := GlobalCausalityToleranceMs * 1000000
	violations := graph.ValidateEdgeConstraints(toleranceNanos)
	if len(violations) > 0 {
		okAll = false
		// Add violations as a result
		dagResult := StepResult{
			Step:    "DAG Validation",
			Call:    "internal",
			Status:  "FAIL",
			Message: fmt.Sprintf("Detected %d DAG constraint violations", len(violations)),
		}
		results = append(results, dagResult)

		// Output detailed violations
		for _, v := range violations {
			fmt.Printf("[DAG Violation] %s: %s\n", v.Type, v.Message)
		}
	}

	// Build span matching index by service.operation
	spanIndex := make(map[string][]trace.Span)
	for _, span := range tr.Spans {
		key := fmt.Sprintf("%s.%s", span.Service, span.Name)
		spanIndex[key] = append(spanIndex[key], span)
	}
	
	// Validate each node in topological order
	topOrder, err := topologicalSort(fs.Graph)
	if err != nil {
		// Should not happen if lint passed, but handle gracefully
		for _, node := range fs.Graph.Nodes {
			results = append(results, StepResult{
				Step: node.ID,
				Call: node.Call,
				Status: "FAIL",
				Message: fmt.Sprintf("DAG topological sort failed: %v", err),
			})
		}
		return results, false
	}
	
	// Track matched spans to avoid double-matching
	usedSpans := make(map[string]bool) // span service:name:startNanos
	
	for _, nodeID := range topOrder {
		node := findNodeByID(fs.Graph, nodeID)
		if node == nil {
			results = append(results, StepResult{
				Step: nodeID, 
				Call: "", 
				Status: "FAIL",
				Message: "Node not found",
			})
			okAll = false
			continue
		}
		
		// Find matching spans for this node
		candidateSpans := spanIndex[node.Call]
		var matchedSpan *trace.Span
		
		for _, span := range candidateSpans {
			spanKey := fmt.Sprintf("%s:%s:%d", span.Service, span.Name, span.StartNanos)
			if !usedSpans[spanKey] {
				matchedSpan = &span
				usedSpans[spanKey] = true
				break
			}
		}
		
		if matchedSpan == nil {
			results = append(results, StepResult{
				Step: node.ID, 
				Call: node.Call, 
				Status: "FAIL", 
				Message: "No matching span found in trace",
			})
			okAll = false
			continue
		}
		
		// Perform causality checking if enabled
		if GlobalCausalityMode != CausalityOff {
			if err := validateCausality(node, matchedSpan, fs.Graph, tr, usedSpans); err != nil {
				results = append(results, StepResult{
					Step: node.ID,
					Call: node.Call,
					Status: "FAIL",
					Message: fmt.Sprintf("Causality validation failed: %v", err),
				})
				okAll = false
				continue
			}
		}
		
		// Basic semantic validation
		var conditions []ConditionResult
		if EnableSemantic {
			// Similar to flow validation - check service operation conditions
			if ops, ok := opIndex[getServiceFromCall(node.Call)]; ok {
				if op, exists := ops[getOperationFromCall(node.Call)]; exists {
					// Create a temporary FlowStep for condition evaluation
					tempStep := spec.FlowStep{
						Step: node.ID,
						Call: node.Call,
						Input: node.Input,
						Output: node.Output,
						Meta: node.Meta,
					}
					conditions, _ = EvaluateConditions(tempStep, op, *matchedSpan, nil)
				}
			}
		}
		
		// Determine overall status based on conditions
		status := "PASS"
		var message string
		if EnableSemantic && len(conditions) > 0 {
			for _, cond := range conditions {
				if cond.Status == "FAIL" {
					status = "FAIL"
					message = "语义校验未通过"
					okAll = false
					break
				}
			}
		}
		
		results = append(results, StepResult{
			Step: node.ID,
			Call: node.Call,
			Status: status,
			Message: message,
			Conditions: conditions,
		})
	}
	
	return results, okAll
}

// validateCausality checks causality constraints for DAG nodes
func validateCausality(node *spec.GraphNode, nodeSpan *trace.Span, graph *spec.GraphSpec, tr *trace.Trace, usedSpans map[string]bool) error {
	// Get predecessor nodes
	predecessors := getPredecessors(node.ID, graph)
	
	for _, predID := range predecessors {
		// Find the span that was matched to this predecessor
		predNode := findNodeByID(graph, predID)
		if predNode == nil {
			continue
		}
		
		// Find the matched span for predecessor
		var predSpan *trace.Span
		for _, span := range tr.Spans {
			spanKey := fmt.Sprintf("%s:%s:%d", span.Service, span.Name, span.StartNanos)
			if usedSpans[spanKey] && span.Service == getServiceFromCall(predNode.Call) && span.Name == getOperationFromCall(predNode.Call) {
				predSpan = &span
				break
			}
		}
		
		if predSpan == nil {
			continue // Predecessor not found, will be caught in its own validation
		}
		
		// Apply causality mode
		switch GlobalCausalityMode {
		case CausalityStrict:
			// Check parent-child relationship
			if !isParentChild(predSpan, nodeSpan) {
				return fmt.Errorf("node %s 应该是 %s 的子节点（strict 模式）", node.ID, predID)
			}
		case CausalityTemporal:
			// Check temporal ordering: predecessor should start before or at the same time as current
			if nodeSpan.StartNanos < predSpan.StartNanos {
				return fmt.Errorf("node %s 开始时间早于前驱节点 %s（temporal 模式）", node.ID, predID)
			}
		}
	}
	
	return nil
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
	
	var result []string
	for len(queue) > 0 {
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
		if edge.To == nodeID {
			preds = append(preds, edge.From)
		}
	}
	return preds
}

func getServiceFromCall(call string) string {
	parts := strings.SplitN(call, ".", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func getOperationFromCall(call string) string {
	parts := strings.SplitN(call, ".", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func isParentChild(parent, child *trace.Span) bool {
	// Check if child has parent span ID that matches parent's span ID
	if parentSpanID, exists := child.Attributes["otlp.parent_span_id"]; exists {
		if spanID, exists := parent.Attributes["otlp.span_id"]; exists {
			return parentSpanID == spanID
		}
	}
	return false
}

// normalize 标准化字符串用于比较
func normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
