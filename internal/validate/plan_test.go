// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func planFixture() (*spec.FlowSpec, map[string]map[string]spec.ServiceOperation, *trace.Trace) {
	f := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "A", Call: "svc.A"}}}
	ops := map[string]map[string]spec.ServiceOperation{"svc": {"A": {Postconditions: map[string]string{"one": "response.status == 200", "two": "response.status == 200"}}}}
	tr := &trace.Trace{Spans: []trace.Span{{Service: "svc", Name: "A", StartNanos: 1, EndNanos: 2, Attributes: map[string]any{"response.status": 200}}}}
	return f, ops, tr
}
func TestContractPlanFrozenConcurrentReuse(t *testing.T) {
	f, ops, tr := planFixture()
	p, err := CompilePlan(f, ops, spec.DefaultValidationConfig())
	if err != nil {
		t.Fatal(err)
	}
	if p.CompiledExpressions() != 1 {
		t.Fatal("duplicate expression compiled twice")
	}
	f.Flow[0].Call = "svc.B"
	ops["svc"]["A"].Postconditions["one"] = "false"
	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, ok := p.ValidateTrace(tr)
			if !ok {
				t.Errorf("shared plan corrupted: %+v", results)
			}
		}()
	}
	wg.Wait()
	next, err := CompilePlan(f, ops, spec.DefaultValidationConfig())
	if err != nil {
		t.Fatal(err)
	}
	if next.Hash() == p.Hash() {
		t.Fatal("changed contract reused old identity")
	}
	if _, ok := next.ValidateTrace(tr); ok {
		t.Fatal("fresh plan ignored changes")
	}
}
func TestContractPlanQualificationAndBudgets(t *testing.T) {
	for _, expr := range []string{"response.status ==", "200", strings.Repeat(" ", input.MaxExpressionBytes) + "true"} {
		f, ops, _ := planFixture()
		ops["svc"]["A"].Postconditions["one"] = expr
		if _, err := CompilePlan(f, ops, spec.DefaultValidationConfig()); err == nil {
			t.Fatalf("accepted %q", expr[:min(len(expr), 30)])
		}
	}
	f, ops, tr := planFixture()
	cfg := spec.DefaultValidationConfig()
	cfg.Limits.MaxCELCost = 1
	p, err := CompilePlan(f, ops, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if results, ok := p.ValidateTrace(tr); ok {
		t.Fatalf("cost overrun passed: %+v", results)
	}
	cfg.Limits.MaxCELCost = 100
	p, err = CompilePlan(f, ops, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if results, ok := p.ValidateTrace(tr); !ok {
		t.Fatalf("positive cost control failed: %+v", results)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := &evaluation{plan: p, ctx: ctx, cancel: cancel}
	if _, phase, err := e.value("response.status == 200", nil); err == nil || phase != "budget" {
		t.Fatal("cancelled request evaluated")
	}
	f.Flow[0].Output = map[string]string{"large": "[1,2,3,4,5,6,7,8,9,10]"}
	op := ops["svc"]["A"]
	op.Postconditions = nil
	ops["svc"]["A"] = op
	cfg.Limits.MaxCELCost = 5
	p, err = CompilePlan(f, ops, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.ValidateTrace(tr); ok {
		t.Fatal("output materialization budget ignored")
	}
	cfg.Limits.MaxSpans = 1
	cfg.Limits.MaxCELCost = 100
	p, err = CompilePlan(f, ops, cfg)
	if err != nil {
		t.Fatal(err)
	}
	tr.Spans = append(tr.Spans, tr.Spans[0])
	if _, ok := p.ValidateTrace(tr); ok {
		t.Fatal("caller-owned trace bypassed span cap")
	}
	f.Flow = append(f.Flow, f.Flow[0])
	cfg.Limits.MaxSteps = 1
	if _, err := CompilePlan(f, ops, cfg); err == nil {
		t.Fatal("step cap ignored")
	}
}

func BenchmarkContractPlan(b *testing.B) {
	f, ops, tr := planFixture()
	b.Run("PrepareAndValidate", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, ok := ValidateAgainstTrace(f, ops, tr, spec.DefaultValidationConfig())
			if !ok {
				b.Fatal("failed")
			}
		}
	})
	p, err := CompilePlan(f, ops, spec.DefaultValidationConfig())
	if err != nil {
		b.Fatal(err)
	}
	b.Run("ReusePlan", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, ok := p.ValidateTrace(tr)
			if !ok {
				b.Fatal("failed")
			}
		}
	})
}
