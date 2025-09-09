package validate

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestBuildCallGraph(t *testing.T) {
	// 创建测试spans
	spans := []trace.Span{
		{
			Attributes: map[string]any{
				"otlp.span_id":  "span1",
				"otlp.trace_id": "trace1",
			},
			Service:    "serviceA",
			Name:       "operationA",
			StartNanos: 1000,
			EndNanos:   2000,
		},
		{
			Attributes: map[string]any{
				"otlp.span_id":        "span2",
				"otlp.trace_id":       "trace1", 
				"otlp.parent_span_id": "span1",
			},
			Service:    "serviceB",
			Name:       "operationB",
			StartNanos: 1500,
			EndNanos:   1800,
		},
	}

	graph, err := BuildCallGraph(spans)
	if err != nil {
		t.Fatalf("BuildCallGraph failed: %v", err)
	}

	// 验证节点数量
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}

	// 验证父子关系
	parentNode := graph.Nodes["span1"]
	childNode := graph.Nodes["span2"]

	if parentNode == nil {
		t.Fatal("Parent node not found")
	}
	if childNode == nil {
		t.Fatal("Child node not found")
	}

	if len(parentNode.Children) != 1 {
		t.Errorf("Expected parent to have 1 child, got %d", len(parentNode.Children))
	}

	if childNode.Parent != parentNode {
		t.Error("Child node parent reference is incorrect")
	}

	// 验证边的创建
	if len(graph.Edges) == 0 {
		t.Error("Expected edges to be created")
	}
}

func TestCheckCausality(t *testing.T) {
	// 创建测试流程规约
	flowSpec := &spec.FlowSpec{
		Flow: []spec.FlowStep{
			{
				Step: "step1",
				Call: "serviceA.operationA",
			},
			{
				Step: "parallel_group",
				Parallel: []spec.FlowStep{
					{
						Step: "step2",
						Call: "serviceB.operationB",
					},
					{
						Step: "step3", 
						Call: "serviceC.operationC",
					},
				},
			},
		},
	}

	// 创建对应的调用图
	graph := &CallGraph{
		Nodes: map[string]*CallNode{
			"span1": {
				SpanID:     "span1",
				Service:    "serviceA",
				Operation:  "operationA",
				StartNanos: 1000,
				EndNanos:   2000,
			},
			"span2": {
				SpanID:     "span2",
				Service:    "serviceB", 
				Operation:  "operationB",
				StartNanos: 2100,
				EndNanos:   2300,
			},
			"span3": {
				SpanID:     "span3",
				Service:    "serviceC",
				Operation:  "operationC", 
				StartNanos: 2150,
				EndNanos:   2250,
			},
		},
		Edges: []*CallEdge{},
	}

	results, allPassed := CheckCausality(flowSpec, graph)

	if !allPassed {
		t.Error("Expected all steps to pass causality check")
	}

	if len(results) != 3 { // 1个常规步骤 + 2个并发步骤
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// 验证步骤状态
	for _, result := range results {
		if result.Status != "PASS" {
			t.Errorf("Expected step %s to PASS, got %s: %s", result.Step, result.Status, result.Message)
		}
	}
}

func TestCheckParallelSteps(t *testing.T) {
	// 并发步骤测试
	parallelSteps := []spec.FlowStep{
		{
			Step: "concurrent1",
			Call: "serviceX.operationX",
		},
		{
			Step: "concurrent2", 
			Call: "serviceY.operationY",
		},
	}

	// 创建调用图，两个操作有重叠的时间窗口（并发）
	graph := &CallGraph{
		Nodes: map[string]*CallNode{
			"spanX": {
				SpanID:     "spanX",
				Service:    "serviceX",
				Operation:  "operationX",
				StartNanos: 1000,
				EndNanos:   2000,
			},
			"spanY": {
				SpanID:     "spanY", 
				Service:    "serviceY",
				Operation:  "operationY",
				StartNanos: 1500, // 重叠时间窗口
				EndNanos:   2500,
			},
		},
		Edges: []*CallEdge{},
	}

	results := checkParallelSteps(parallelSteps, graph)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// 两个并发步骤都应该找到匹配的span
	for _, result := range results {
		if result.Status != "PASS" {
			t.Errorf("Expected step %s to PASS, got %s: %s", result.Step, result.Status, result.Message)
		}
	}
}

func TestCheckSingleStep(t *testing.T) {
	step := spec.FlowStep{
		Step: "test_step",
		Call: "testService.testOperation",
	}

	graph := &CallGraph{
		Nodes: map[string]*CallNode{
			"test_span": {
				SpanID:     "test_span",
				Service:    "testService",
				Operation:  "testOperation",
				StartNanos: 1000,
				EndNanos:   2000,
			},
		},
		Edges: []*CallEdge{},
	}

	result := checkSingleStep(step, graph)

	if result.Status != "PASS" {
		t.Errorf("Expected PASS, got %s: %s", result.Status, result.Message)
	}

	if result.Step != "test_step" {
		t.Errorf("Expected step name 'test_step', got '%s'", result.Step)
	}

	if result.Call != "testService.testOperation" {
		t.Errorf("Expected call 'testService.testOperation', got '%s'", result.Call)
	}
}

func TestCheckSingleStepNotFound(t *testing.T) {
	step := spec.FlowStep{
		Step: "missing_step",
		Call: "missingService.missingOperation",
	}

	graph := &CallGraph{
		Nodes: map[string]*CallNode{
			"other_span": {
				SpanID:     "other_span",
				Service:    "otherService",
				Operation:  "otherOperation",
				StartNanos: 1000,
				EndNanos:   2000,
			},
		},
		Edges: []*CallEdge{},
	}

	result := checkSingleStep(step, graph)

	if result.Status != "FAIL" {
		t.Errorf("Expected FAIL, got %s", result.Status)
	}

	if result.Step != "missing_step" {
		t.Errorf("Expected step name 'missing_step', got '%s'", result.Step)
	}
}

func TestGetSpanIDFallback(t *testing.T) {
	// 测试span ID提取的各种情况
	tests := []struct {
		name     string
		span     trace.Span
		expected string
	}{
		{
			name: "otlp.span_id attribute", 
			span: trace.Span{
				Attributes: map[string]any{
					"otlp.span_id": "test_span_123",
				},
			},
			expected: "test_span_123",
		},
		{
			name: "fallback to service:name:start",
			span: trace.Span{
				Service:    "fallbackService",
				Name:       "fallbackOp",
				StartNanos: 12345,
				Attributes: map[string]any{},
			},
			expected: "fallbackService:fallbackOp:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSpanID(tt.span)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNormalizeComparison(t *testing.T) {
	// 测试normalize函数的字符串标准化
	tests := []struct {
		input1   string
		input2   string
		shouldMatch bool
	}{
		{"ServiceA", "servicea", true},
		{"  Service B  ", "service b", true},
		{"OperationName", "operationName", true},
		{"Different", "Services", false},
	}

	for _, tt := range tests {
		normalized1 := normalize(tt.input1)
		normalized2 := normalize(tt.input2)
		matches := (normalized1 == normalized2)
		
		if matches != tt.shouldMatch {
			t.Errorf("normalize('%s') vs normalize('%s'): expected match=%t, got match=%t", 
				tt.input1, tt.input2, tt.shouldMatch, matches)
		}
	}
}