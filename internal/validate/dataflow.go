// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

func variableValue(path string, vars map[string]any) (any, bool) {
	var value any = vars
	for _, part := range strings.Split(path, ".") {
		switch current := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = current[part]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(current) {
				return nil, false
			}
			value = current[index]
		default:
			return nil, false
		}
	}
	return value, true
}

// Unknown input values remain CEL errors, never nonempty placeholder strings.
// Conditions which do not read an external input can still be evaluated.
func resolveInput(input any, vars map[string]any) any {
	switch value := input.(type) {
	case map[string]any:
		result := map[string]any{}
		for key, child := range value {
			result[key] = resolveInput(child, vars)
		}
		return result
	case []any:
		result := make([]any, len(value))
		for i, child := range value {
			result[i] = resolveInput(child, vars)
		}
		return result
	case string:
		matches := varRefRe.FindAllStringSubmatchIndex(value, -1)
		if len(matches) == 0 {
			return value
		}
		var text strings.Builder
		previous := 0
		for _, match := range matches {
			name := value[match[2]:match[3]]
			resolved, ok := variableValue(name, vars)
			if !ok {
				return types.NewErr("unresolved input variable ${%s}", name)
			}
			if len(matches) == 1 && match[0] == 0 && match[1] == len(value) {
				return resolved
			}
			switch resolved.(type) {
			case map[string]any, []any:
				return types.NewErr("cannot embed object variable ${%s} in text", name)
			}
			text.WriteString(value[previous:match[0]])
			_, _ = fmt.Fprint(&text, resolved)
			previous = match[1]
		}
		text.WriteString(value[previous:])
		return text.String()
	default:
		return input
	}
}

func nativeValue(value ref.Val) (any, error) {
	switch current := value.(type) {
	case traits.Mapper:
		result := map[string]any{}
		iterator := current.Iterator()
		for iterator.HasNext() == types.True {
			key := iterator.Next()
			name, ok := key.Value().(string)
			if !ok {
				return nil, fmt.Errorf("output maps require string keys")
			}
			child, err := nativeValue(current.Get(key))
			if err != nil {
				return nil, err
			}
			result[name] = child
		}
		return result, nil
	case traits.Lister:
		result := []any{}
		iterator := current.Iterator()
		for iterator.HasNext() == types.True {
			child, err := nativeValue(iterator.Next())
			if err != nil {
				return nil, err
			}
			result = append(result, child)
		}
		return result, nil
	default:
		if types.IsError(value) || types.IsUnknown(value) {
			return nil, fmt.Errorf("output was not evaluated: %v", value)
		}
		return value.Value(), nil
	}
}

func evaluateStep(result StepResult, step spec.FlowStep, span trace.Span, ops map[string]map[string]spec.ServiceOperation, vars map[string]any) (StepResult, map[string]any) {
	if result.Status != "PASS" || !EnableSemantic {
		return result, nil
	}
	svc, operation, _ := splitCall(step.Call)
	if op, exists := ops[svc][operation]; exists {
		var passed bool
		result.Conditions, passed = EvaluateConditions(step, op, span, vars)
		if !passed {
			result.Status, result.Message = "FAIL", "semantic validation failed"
			return result, nil
		}
	}
	env, err := buildEvalEnvForStep(step, span, vars)
	if err != nil {
		result.Status, result.Message = "FAIL", err.Error()
		return result, nil
	}
	exports := map[string]any{}
	names := make([]string, 0, len(step.Output))
	for name := range step.Output {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value, phase, err := evalCELValue(step.Output[name], env)
		if err == nil {
			exports[name], err = nativeValue(value)
		}
		if err != nil {
			result.Status, result.Message = "FAIL", fmt.Sprintf("output %s evaluation failed (%s): %v", name, phase, err)
			return result, nil
		}
	}
	return result, exports
}

func evaluateFlowMatches(matches []matchedFlowStep, ops map[string]map[string]spec.ServiceOperation) []StepResult {
	var results []StepResult
	vars := map[string]any{}
	for begin := 0; begin < len(matches); {
		end := begin + 1
		for end < len(matches) && matches[end].stage == matches[begin].stage {
			end++
		}
		group := make([]StepResult, end-begin)
		exports, owners, conflicts := map[string]any{}, map[string]int{}, map[string]bool{}
		for i := begin; i < end; i++ {
			match := matches[i]
			result := match.result
			produced := map[string]any{}
			if match.node != nil {
				node := match.node
				span := trace.Span{Service: node.Service, Name: node.Operation, StartNanos: node.StartNanos, EndNanos: node.EndNanos, Attributes: node.Attributes}
				result, produced = evaluateStep(result, match.step, span, ops, vars)
			}
			group[i-begin] = result
			for name, value := range produced {
				if owner, exists := owners[name]; exists {
					conflicts[name] = true
					group[owner].Status, group[owner].Message = "FAIL", "ambiguous parallel output variable "+name
					group[i-begin].Status, group[i-begin].Message = "FAIL", "ambiguous parallel output variable "+name
				}
				owners[name], exports[name] = i-begin, value
			}
		}
		for name, value := range exports {
			if !conflicts[name] && group[owners[name]].Status == "PASS" {
				vars[name] = value
			}
		}
		results = append(results, group...)
		begin = end
	}
	return results
}

func graphAncestors(graph *spec.GraphSpec, id string) map[string]bool {
	ancestors := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		for _, pred := range getPredecessors(id, graph) {
			if !ancestors[pred] {
				ancestors[pred] = true
				visit(pred)
			}
		}
	}
	visit(id)
	return ancestors
}

func graphVariables(graph *spec.GraphSpec, id string, order []string, exports map[string]map[string]any) (map[string]any, error) {
	ancestors := graphAncestors(graph, id)
	vars, writers := map[string]any{}, map[string]string{}
	for _, pred := range order {
		if !ancestors[pred] {
			continue
		}
		for name, value := range exports[pred] {
			if writer, exists := writers[name]; exists && !graphAncestors(graph, pred)[writer] {
				return nil, fmt.Errorf("ambiguous output variable %s from independent predecessors", name)
			}
			vars[name], writers[name] = value, pred
		}
	}
	return vars, nil
}
