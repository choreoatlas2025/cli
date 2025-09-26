// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/choreoatlas2025/cli/internal/spec"
)

// LintIssue 表示静态检查发现的问题
type LintIssue struct {
	Level string // "ERROR" or "WARN"
	Msg   string
}

var varRefRe = regexp.MustCompile(`\$\{\s*([a-zA-Z_][\w\-\.]*)\s*\}`)

// LintFlow 对流程规约进行静态检查
func LintFlow(flowPath string, fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation) ([]LintIssue, error) {
	var issues []LintIssue

	// 1) 基本结构检查
	if fs.Info.Title == "" {
		issues = append(issues, LintIssue{"WARN", "info.title is empty"})
	}
	
	// Check format compatibility
	if len(fs.Flow) == 0 && fs.Graph == nil {
		return append(issues, LintIssue{"ERROR", "either flow or graph must be specified"}), nil
	}
	if len(fs.Flow) > 0 && fs.Graph != nil {
		return append(issues, LintIssue{"ERROR", "cannot specify both 'flow' and 'graph' - please choose one format"}), nil
	}
	
	// Route to appropriate linting based on format
	if fs.IsGraphMode() {
		return lintGraph(fs, opIndex)
	}
	return lintFlow(fs, opIndex)
}

// lintFlow handles traditional flow format linting
func lintFlow(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation) ([]LintIssue, error) {
    var issues []LintIssue

	// Add warning for legacy flow format
	issues = append(issues, LintIssue{"WARN", "Using legacy flow format. Graph (DAG) format is recommended for better expressiveness and validation."})

	if len(fs.Flow) == 0 {
		return append(issues, LintIssue{"ERROR", "flow is empty"}), nil
	}

	// 2) 步骤唯一性 & 调用合法性检查
	stepNames := map[string]struct{}{}
	var allSteps []spec.FlowStep
	
	// 收集所有步骤（包括并发步骤）
	for _, st := range fs.Flow {
		allSteps = append(allSteps, st)
		if len(st.Parallel) > 0 {
			allSteps = append(allSteps, st.Parallel...)
		}
	}
	
    for i, st := range allSteps {
		if st.Step == "" {
			issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("step #%d is missing step name", i+1)})
		}
		if _, ok := stepNames[st.Step]; ok {
			issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("duplicate step name: %s", st.Step)})
		}
		stepNames[st.Step] = struct{}{}

		// 跳过只有并发子步骤的父步骤的call检查
		if st.Call == "" && len(st.Parallel) > 0 {
			continue
		}

		svc, op, err := splitCall(st.Call)
		if err != nil {
			issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("step=%s invalid call: %v", st.Step, err)})
			continue
		}
		ops, ok := opIndex[svc]
		if !ok {
			issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("step=%s references undeclared service: %s", st.Step, svc)})
			continue
		}
        if _, ok := ops[op]; !ok {
            issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("step=%s references non-existent operation %s in service %s", st.Step, op, svc)})
        }

        // 输入键检查：禁止将遥测属性直接放入 FlowSpec.input
        if len(st.Input) > 0 {
            if bad := findTelemetryKeys(st.Input); len(bad) > 0 {
                issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("step=%s input contains telemetry keys not allowed in FlowSpec.input: %s", st.Step, strings.Join(bad, ", "))})
            }
        }
    }

	// 3) 变量引用连贯性检查（简单版）
	// 按步骤顺序，前置步骤输出的 token 可被后续步骤引用
	knownVars := map[string]struct{}{}

	// 预定义一些常见的初始变量（通常从请求或外部输入获得）
	initialVars := []string{"customerId", "orderItems", "totalAmount", "userId", "requestId"}
	for _, v := range initialVars {
		knownVars[v] = struct{}{}
	}

	for _, st := range fs.Flow {
		// 处理并发步骤
		if len(st.Parallel) > 0 {
			// 并发步骤前的变量对所有并发子步骤都可见
			parallelVars := map[string]struct{}{}
			for outVar := range knownVars {
				parallelVars[outVar] = struct{}{}
			}
			
			// 检查并发子步骤的变量依赖
			for _, pst := range st.Parallel {
				for _, v := range collectVarRefs(pst.Input) {
					rootVar := strings.SplitN(v, ".", 2)[0]
					if _, ok := parallelVars[rootVar]; !ok {
						issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("step=%s references unknown variable ${%s}", pst.Step, v)})
					}
				}
			}
			
			// 并发步骤的输出在并发完成后才可用
			for _, pst := range st.Parallel {
				for outVar := range pst.Output {
					knownVars[outVar] = struct{}{}
				}
			}
		} else {
			// 普通步骤的变量检查
			for _, v := range collectVarRefs(st.Input) {
				// 对于嵌套变量引用（如 orderResponse.items），只检查根变量（orderResponse）
				rootVar := strings.SplitN(v, ".", 2)[0]
				if _, ok := knownVars[rootVar]; !ok {
					issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("step=%s references unknown variable ${%s}", st.Step, v)})
				}
			}
			// 将本步骤 output 的 key 视为新的变量名
			for outVar := range st.Output {
				knownVars[outVar] = struct{}{}
			}
		}
	}

	return issues, nil
}

// splitCall 解析服务调用格式
func splitCall(call string) (service string, operation string, err error) {
	parts := strings.SplitN(strings.TrimSpace(call), ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New("call must be in format 'serviceAlias.operationId'")
	}
	return parts[0], parts[1], nil
}

// collectVarRefs 从任意值中收集变量引用
func collectVarRefs(v any) []string {
	var out []string
	switch t := v.(type) {
	case string:
		ms := varRefRe.FindAllStringSubmatch(t, -1)
		for _, m := range ms {
			out = append(out, m[1])
		}
	case map[string]any:
		for _, vv := range t {
			out = append(out, collectVarRefs(vv)...)
		}
	case []any:
		for _, vv := range t {
			out = append(out, collectVarRefs(vv)...)
		}
	}
	return unique(out)
}

// lintGraph handles DAG format linting
func lintGraph(fs *spec.FlowSpec, opIndex map[string]map[string]spec.ServiceOperation) ([]LintIssue, error) {
    var issues []LintIssue
	
	if fs.Graph == nil {
		return append(issues, LintIssue{"ERROR", "graph is empty"}), nil
	}
	
	// 1) Validate basic DAG structure (cycles, connectivity)
	if err := fs.Graph.ValidateGraphStructure(); err != nil {
		issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("DAG structure validation failed: %v", err)})
		return issues, nil // Stop here if structure is invalid
	}
	
	// 2) Node call validation (similar to flow step validation)
    for _, node := range fs.Graph.Nodes {
		svc, op, err := splitCall(node.Call)
		if err != nil {
			issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("node=%s has invalid call: %v", node.ID, err)})
			continue
		}
		ops, ok := opIndex[svc]
		if !ok {
			issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("node=%s references undeclared service: %s", node.ID, svc)})
			continue
		}
        if _, ok := ops[op]; !ok {
            issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("node=%s references non-existent operation %s in service %s", node.ID, op, svc)})
        }

        // 输入键检查：禁止遥测属性出现在输入中
        if len(node.Input) > 0 {
            if bad := findTelemetryKeys(node.Input); len(bad) > 0 {
                issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("node=%s input contains telemetry keys not allowed in FlowSpec.input: %s", node.ID, strings.Join(bad, ", "))})
            }
        }
    }
	
	// 3) Variable flow validation for DAG
	if err := validateVariableFlow(fs.Graph); err != nil {
		issues = append(issues, LintIssue{"ERROR", fmt.Sprintf("Variable flow validation failed: %v", err)})
	}
	
	return issues, nil
}

// findTelemetryKeys 检查输入映射中是否出现明显的遥测前缀键
// 允许的顶层键：path, query, headers, body
// 当在 body 下出现以 http./otel./span. 前缀的键，或顶层直接以这些前缀开头的键，则视为不合法
func findTelemetryKeys(input map[string]any) []string {
    var out []string
    // 允许的顶层键
    allowed := map[string]struct{}{"path": {}, "query": {}, "headers": {}, "body": {}}

    // 收集顶层非法键
    for k := range input {
        if _, ok := allowed[k]; !ok {
            if hasTelemetryPrefix(k) {
                out = append(out, k)
            }
        }
    }
    // 检查 body 下的键
    if b, ok := input["body"].(map[string]any); ok {
        for k := range b {
            if hasTelemetryPrefix(k) {
                out = append(out, "body."+k)
            }
        }
    }
    return unique(out)
}

func hasTelemetryPrefix(k string) bool {
    lower := strings.ToLower(k)
    return strings.HasPrefix(lower, "http.") || strings.HasPrefix(lower, "otel.") || strings.HasPrefix(lower, "span.")
}

// validateVariableFlow checks that variables flow correctly through the DAG
func validateVariableFlow(graph *spec.GraphSpec) error {
	// Ensure edges are built from depends field
	graph.EnsureEdges()

	// Build adjacency list and reverse adjacency list
	adj := make(map[string][]string)
	radj := make(map[string][]string) // reverse adjacency for dependency tracking

	for _, edge := range graph.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
		radj[edge.To] = append(radj[edge.To], edge.From)
	}
	
	// Build node map for easy lookup
	nodeMap := make(map[string]spec.GraphNode)
	for _, node := range graph.Nodes {
		nodeMap[node.ID] = node
	}
	
	// For each node, check that all variable references can be satisfied by predecessor nodes
	for _, node := range graph.Nodes {
		// Collect all variables this node references
		requiredVars := collectVarRefs(node.Input)
		
		// Find all predecessor nodes through DFS
		availableVars := make(map[string]bool)
		
		// Pre-seed with initial variables that are typically available at start
		initialVars := []string{"customerId", "orderItems", "totalAmount", "userId", "requestId"}
		for _, v := range initialVars {
			availableVars[v] = true
		}
		
		// Collect variables from all reachable predecessor nodes
		visited := make(map[string]bool)
		var dfs func(nodeID string)
		dfs = func(nodeID string) {
			if visited[nodeID] {
				return
			}
			visited[nodeID] = true
			
			// Add variables produced by this node
			if predNode, exists := nodeMap[nodeID]; exists {
				for outVar := range predNode.Output {
					availableVars[outVar] = true
				}
			}
			
			// Recurse to predecessor nodes
			for _, pred := range radj[nodeID] {
				dfs(pred)
			}
		}
		
		// Start DFS from all direct predecessors of current node
		for _, pred := range radj[node.ID] {
			dfs(pred)
		}
		
		// Check if all required variables are available
		for _, requiredVar := range requiredVars {
			rootVar := strings.SplitN(requiredVar, ".", 2)[0]
			if !availableVars[rootVar] {
				return fmt.Errorf("node %s references variable ${%s} that is not available from predecessor nodes", node.ID, requiredVar)
			}
		}
	}
	
	return nil
}

// unique 去重并排序字符串切片
func unique(in []string) []string {
	if len(in) == 0 {
		return in
	}
	m := map[string]struct{}{}
	for _, s := range in {
		m[s] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
