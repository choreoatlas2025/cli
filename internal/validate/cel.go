// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/cel-go/common/types"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

// 条件结果
type ConditionResult struct {
	Kind    string `json:"kind"` // "pre" | "post"
	Name    string `json:"name"`
	Expr    string `json:"expr"`
	Status  string `json:"status"`  // "PASS" | "FAIL" | "SKIP"
	Message string `json:"message"` // 失败/跳过原因
}

// 将 FlowSpec 的 input + span.attributes 投影为 CEL 环境可用的变量
// 约定：
// - request: declared input, with typed references resolved from preceding outputs
// - response: 从 span.attributes 映射，优先使用 response.status，再使用 HTTP 状态码
// - span: { name, service, attributes }
// - vars: validated outputs visible to this step
func buildEvalEnvForStep(step spec.FlowStep, sp trace.Span, vars map[string]any) (map[string]any, error) {
	// Resolve declared input without treating missing variables as literal strings.
	request := map[string]any{}
	if step.Input != nil {
		input := resolveInput(step.Input, vars).(map[string]any)
		structured := false
		for _, key := range []string{"body", "path", "query", "headers"} {
			if _, ok := input[key]; ok {
				structured = true
			}
		}
		if structured {
			request = input
		} else {
			request["body"] = input
		}
	}

	// 响应投影：尽量从 attributes 推断出 response.status / response.body
	response := map[string]any{}
	// 提取 status
	statusKeys := []string{"response.status", "http.response.status_code", "http.status_code", "statusCode"}
	var status any
	for _, k := range statusKeys {
		if v, ok := sp.Attributes[k]; ok {
			status = v
			break
		}
	}
	if status != nil {
		response["status"] = status
	} else {
		response["status"] = 0 // 未知
	}
	// body：优先 attributes["response.body"]；否则用整个 attributes 兜底
	if b, ok := sp.Attributes["response.body"]; ok {
		response["body"] = b
	} else {
		response["body"] = sp.Attributes
	}

	span := map[string]any{
		"name":       sp.Name,
		"service":    sp.Service,
		"attributes": sp.Attributes,
	}

	return map[string]any{
		"request":  request,
		"response": response,
		"span":     span,
		"vars":     vars,
	}, nil
}

// 简单规范化表达式：支持 foo =~ /re/ 语法，转为 foo.matches("re")
var reLike = regexp.MustCompile(`\s*=~\s*/([^/]+)/`)

func normalizeExpr(e string) string {
	// 将 x =~ /abc/ 替换为 x.matches("abc")
	// 注意：此实现是简化版，不支持包含斜杠转义的复杂正则
	return reLike.ReplaceAllStringFunc(e, func(m string) string {
		sub := reLike.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		re := sub[1]
		return fmt.Sprintf(`.matches("%s")`, strings.ReplaceAll(re, `"`, `\"`))
	})
}

func (e *evaluation) boolean(expr string, envVars map[string]any) (bool, string, error) {
	out, phase, err := e.value(expr, envVars)
	if err != nil {
		return false, phase, err
	}
	if out.Type() == types.BoolType {
		return out.Value().(bool), "", nil
	}
	// 动态类型时再尝试强转
	if b, ok := out.Value().(bool); ok {
		return b, "", nil
	}
	return false, "type", fmt.Errorf("expr result not bool: %T", out.Value())
}

// conditions evaluates a step's pre/postconditions in stable name order.
// Declared conditions must evaluate successfully to a boolean. Evaluation
// errors are failures, not skipped evidence.
func (e *evaluation) conditions(
	step spec.FlowStep,
	op spec.ServiceOperation,
	sp trace.Span,
	vars map[string]any,
) ([]ConditionResult, bool) {

	results := []ConditionResult{}
	passAll := true

	envVars, _ := buildEvalEnvForStep(step, sp, vars)

	// 预条件
	for _, name := range sortedKeys(op.Preconditions) {
		expr := op.Preconditions[name]
		ok, phase, err := e.boolean(expr, envVars)
		cr := ConditionResult{Kind: "pre", Name: name, Expr: expr}
		if err != nil {
			cr.Status = "FAIL"
			cr.Message = fmt.Sprintf("CEL %s error: %v", phase, err)
			passAll = false
		} else if ok {
			cr.Status = "PASS"
		} else {
			cr.Status = "FAIL"
			cr.Message = "result is false"
			passAll = false
		}
		results = append(results, cr)
	}

	// 后置条件
	for _, name := range sortedKeys(op.Postconditions) {
		expr := op.Postconditions[name]
		ok, phase, err := e.boolean(expr, envVars)
		cr := ConditionResult{Kind: "post", Name: name, Expr: expr}
		if err != nil {
			cr.Status = "FAIL"
			cr.Message = fmt.Sprintf("CEL %s error: %v", phase, err)
			passAll = false
		} else if ok {
			cr.Status = "PASS"
		} else {
			cr.Status = "FAIL"
			cr.Message = "result is false"
			passAll = false
		}
		results = append(results, cr)
	}

	return results, passAll
}
