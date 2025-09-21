// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

// 条件结果
type ConditionResult struct {
	Kind    string `json:"kind"`    // "pre" | "post"
	Name    string `json:"name"`
	Expr    string `json:"expr"`
	Status  string `json:"status"`  // "PASS" | "FAIL" | "SKIP"
	Message string `json:"message"` // 失败/跳过原因
}

// 将 FlowSpec 的 input + span.attributes 投影为 CEL 环境可用的变量
// 约定：
// - request: 来自 step.input（会做 ${var} 的占位保留，不做替换以免误导，可后续扩展变量解引用）
// - response: 从 span.attributes 映射（response.status 优先取：response.status|http.status_code|statusCode）
// - span: { name, service, attributes }
// - vars: 从前序步骤输出收集（可选，当前为占位）
func buildEvalEnvForStep(step spec.FlowStep, sp trace.Span, vars map[string]any) (map[string]any, error) {
	// request 直接采用 FlowSpec 中的 input 原样
	request := map[string]any{}
	if step.Input != nil {
		request["body"] = step.Input // 约定 input 即 body，满足大多数 REST 场景
	}

	// 响应投影：尽量从 attributes 推断出 response.status / response.body
	response := map[string]any{}
	// 提取 status
	statusKeys := []string{"response.status", "http.status_code", "statusCode"}
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

func evalCELBool(expr string, envVars map[string]any) (bool, string, error) {
	e := normalizeExpr(expr)

	// 使用新版 CEL API 创建环境
	celEnv, err := cel.NewEnv(
		cel.Variable("request", cel.DynType),
		cel.Variable("response", cel.DynType),
		cel.Variable("span", cel.DynType),
		cel.Variable("vars", cel.DynType),
	)
	if err != nil {
		return false, "", fmt.Errorf("create cel env: %w", err)
	}

	ast, issues := celEnv.Compile(e)
	if issues != nil && issues.Err() != nil {
		return false, "compile", issues.Err()
	}

	prg, err := celEnv.Program(ast)
	if err != nil {
		return false, "program", err
	}

	out, _, err := prg.Eval(envVars)
	if err != nil {
		return false, "runtime", err
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

// EvaluateConditions 对某一步骤的 pre/postconditions 进行求值
// 说明：编译错误/不支持表达式 -> SKIP，不计为失败
func EvaluateConditions(
	step spec.FlowStep,
	op spec.ServiceOperation,
	sp trace.Span,
	vars map[string]any,
) ([]ConditionResult, bool) {

	results := []ConditionResult{}
	passAll := true

	envVars, _ := buildEvalEnvForStep(step, sp, vars)

	// 预条件
	for name, expr := range op.Preconditions {
		ok, phase, err := evalCELBool(expr, envVars)
		cr := ConditionResult{Kind: "pre", Name: name, Expr: expr}
		if err != nil {
			cr.Status = "SKIP"
			cr.Message = fmt.Sprintf("unsupported or compilation failed (%s): %v", phase, err)
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
	for name, expr := range op.Postconditions {
		ok, phase, err := evalCELBool(expr, envVars)
		cr := ConditionResult{Kind: "post", Name: name, Expr: expr}
		if err != nil {
			cr.Status = "SKIP"
			cr.Message = fmt.Sprintf("unsupported or compilation failed (%s): %v", phase, err)
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