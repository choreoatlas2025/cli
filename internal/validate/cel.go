// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"fmt"
	"github.com/choreoatlas2025/cli/internal/verdict"
	"regexp"
	"strings"

	"github.com/google/cel-go/common/types"

	"github.com/choreoatlas2025/cli/internal/evidence"
	"github.com/choreoatlas2025/cli/internal/spec"
)

type ConditionResult = verdict.ConditionResult

// Declarations are expected values; request/response are captured observations.
func buildEvalEnvironment(step spec.FlowStep, projected evidence.Projection, vars map[string]any) map[string]any {
	expected := map[string]any{}
	if step.Input != nil {
		input := resolveInput(step.Input, vars).(map[string]any)
		structured := false
		for _, key := range []string{"body", "path", "query", "headers"} {
			if _, ok := input[key]; ok {
				structured = true
			}
		}
		if structured {
			expected = input
		} else {
			expected["body"] = input
		}
	}
	return map[string]any{"expected": expected, "request": projected.Request, "response": projected.Response, "span": projected.Span, "vars": vars}
}

// Evaluation errors remain failures. Missing keys and unresolved declared
// references are insufficient evidence; other CEL errors are execution errors.
func evaluationIssue(err error) verdict.IssueKind {
	if strings.Contains(err.Error(), "no such key:") || strings.Contains(err.Error(), "unresolved input variable") {
		return verdict.MissingEvidence
	}
	return verdict.ExecutionError
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
	op spec.ServiceOperation,
	envVars map[string]any,
) ([]ConditionResult, bool) {

	results := []ConditionResult{}
	passAll := true

	// 预条件
	for _, name := range sortedKeys(op.Preconditions) {
		expr := op.Preconditions[name]
		ok, phase, err := e.boolean(expr, envVars)
		cr := ConditionResult{Kind: "pre", Name: name, Expr: expr}
		if err != nil {
			cr.Status = "FAIL"
			cr.Issue = evaluationIssue(err)
			cr.Message = fmt.Sprintf("CEL %s error: %v", phase, err)
			passAll = false
		} else if ok {
			cr.Status = "PASS"
		} else {
			cr.Status = "FAIL"
			cr.Message = "result is false"
			cr.Issue = verdict.RuleViolation
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
			cr.Issue = evaluationIssue(err)
			cr.Message = fmt.Sprintf("CEL %s error: %v", phase, err)
			passAll = false
		} else if ok {
			cr.Status = "PASS"
		} else {
			cr.Status = "FAIL"
			cr.Message = "result is false"
			cr.Issue = verdict.RuleViolation
			passAll = false
		}
		results = append(results, cr)
	}

	return results, passAll
}
