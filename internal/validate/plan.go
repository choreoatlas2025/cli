// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/choreoatlas2025/cli/internal/evidence"
	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"gopkg.in/yaml.v3"
)

type compiledExpression struct {
	program cel.Program
	output  *cel.Type
}

// ContractPlan owns a frozen contract and immutable compiled programs. Rebuild
// it for changed input/configuration; requests never share mutable evaluation state.
type ContractPlan struct {
	flow       *spec.FlowSpec
	operations map[string]map[string]spec.ServiceOperation
	config     spec.ValidationConfig
	programs   map[string]compiledExpression
	hash       string
}

func (p *ContractPlan) Hash() string             { return p.hash }
func (p *ContractPlan) CompiledExpressions() int { return len(p.programs) }

func CompilePlan(flow *spec.FlowSpec, ops map[string]map[string]spec.ServiceOperation, config spec.ValidationConfig) (*ContractPlan, error) {
	if flow == nil {
		return nil, fmt.Errorf("invalid contract: nil flow")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config.Limits = config.Limits.Normalized()
	if flow.GetStepsCount() > config.Limits.MaxSteps {
		return nil, fmt.Errorf("invalid contract: step count exceeds %d", config.Limits.MaxSteps)
	}
	flowBytes, err := yaml.Marshal(flow)
	if err != nil {
		return nil, err
	}
	frozen, err := spec.ParseFlowSpec(flowBytes)
	if err != nil {
		return nil, err
	}
	opBytes, err := yaml.Marshal(ops)
	if err != nil {
		return nil, err
	}
	owned := map[string]map[string]spec.ServiceOperation{}
	if err := yaml.Unmarshal(opBytes, &owned); err != nil {
		return nil, err
	}
	p := &ContractPlan{flow: frozen, operations: owned, config: config, programs: map[string]compiledExpression{}}
	identity, err := json.Marshal(struct {
		Flow, Operations, EvidenceSemantics string
		Config                              spec.ValidationConfig
	}{string(flowBytes), string(opBytes), evidence.SemanticsVersion, config})
	if err != nil {
		return nil, err
	}
	p.hash = fmt.Sprintf("sha256:%x", sha256.Sum256(identity))
	if !config.Semantic {
		return p, nil
	}
	env, err := cel.NewEnv(cel.Variable("expected", cel.DynType), cel.Variable("request", cel.DynType), cel.Variable("response", cel.DynType), cel.Variable("span", cel.DynType), cel.Variable("vars", cel.DynType), cel.ParserRecursionLimit(100), cel.ParserExpressionSizeLimit(input.MaxExpressionBytes))
	if err != nil {
		return nil, err
	}
	count := 0
	bytesTotal := 0
	compileContext, cancel := context.WithTimeout(context.Background(), time.Duration(config.Limits.TimeoutMs)*time.Millisecond)
	defer cancel()
	compile := func(expr string, boolean bool) error {
		if err := compileContext.Err(); err != nil {
			return fmt.Errorf("invalid contract preparation budget: %w", err)
		}
		count++
		bytesTotal += len(expr)
		if count > input.MaxExpressions || len(expr) > input.MaxExpressionBytes || bytesTotal > input.MaxExpressionTotalBytes {
			return fmt.Errorf("invalid CEL contract: expression count or length exceeds limit")
		}
		key := normalizeExpr(expr)
		compiled, exists := p.programs[key]
		if !exists {
			ast, issues := env.Compile(key)
			if issues != nil && issues.Err() != nil {
				return fmt.Errorf("invalid CEL contract: %w", issues.Err())
			}
			program, err := env.Program(ast, cel.CostLimit(config.Limits.MaxCELCost), cel.InterruptCheckFrequency(100))
			if err != nil {
				return fmt.Errorf("invalid CEL program: %w", err)
			}
			compiled = compiledExpression{program: program, output: ast.OutputType()}
			p.programs[key] = compiled
		}
		if boolean && !compiled.output.IsExactType(cel.BoolType) && !compiled.output.IsExactType(cel.DynType) {
			return fmt.Errorf("invalid CEL condition: result must be bool, got %s", compiled.output)
		}
		return nil
	}
	aliases := sortedKeys(owned)
	for _, alias := range aliases {
		for _, name := range sortedKeys(owned[alias]) {
			op := owned[alias][name]
			for _, conditions := range []map[string]string{op.Preconditions, op.Postconditions} {
				for _, key := range sortedKeys(conditions) {
					if err := compile(conditions[key], true); err != nil {
						return nil, fmt.Errorf("%s.%s condition %s: %w", alias, name, key, err)
					}
				}
			}
		}
	}
	for _, step := range frozen.CallSteps() {
		for _, name := range sortedKeys(step.Output) {
			if err := compile(step.Output[name], false); err != nil {
				return nil, fmt.Errorf("step %s output %s: %w", step.Step, name, err)
			}
		}
	}
	return p, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type evaluation struct {
	plan   *ContractPlan
	ctx    context.Context
	cancel context.CancelFunc
	cost   uint64
}

func (p *ContractPlan) ValidateTrace(tr *trace.Trace) ([]StepResult, bool) {
	if tr == nil {
		return []StepResult{{Step: "trace-input", Status: "FAIL", Message: "nil trace"}}, false
	}
	if len(tr.Spans) > p.config.Limits.MaxSpans {
		return []StepResult{{Step: "trace-budget", Call: "internal", Status: "FAIL", Message: "trace span count exceeds configured limit"}}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.config.Limits.TimeoutMs)*time.Millisecond)
	defer cancel()
	eval := &evaluation{plan: p, ctx: ctx, cancel: cancel}
	results, _ := validateAgainstTrace(p.flow, p.operations, tr, p.config, eval)
	if err := ctx.Err(); err != nil {
		results = append(results, StepResult{Step: "validation-budget", Call: "internal", Status: "FAIL", Message: err.Error()})
	}
	return results, AllStepsPassed(results)
}

func (e *evaluation) value(expr string, vars map[string]any) (ref.Val, string, error) {
	if err := e.ctx.Err(); err != nil {
		return nil, "budget", err
	}
	compiled, exists := e.plan.programs[normalizeExpr(expr)]
	if !exists {
		return nil, "plan", fmt.Errorf("stale contract plan: expression was not qualified")
	}
	value, details, err := compiled.program.ContextEval(e.ctx, vars)
	if details != nil && details.ActualCost() != nil {
		cost := *details.ActualCost()
		if cost > e.plan.config.Limits.MaxCELCost-e.cost {
			e.cancel()
			return nil, "budget", fmt.Errorf("CEL invocation cost limit exceeded")
		}
		e.cost += cost
	}
	if err != nil {
		return nil, "runtime", err
	}
	return value, "", nil
}
