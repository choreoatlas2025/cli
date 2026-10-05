// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package result owns shared decisions and measurements, independent of renderers.
package result

import (
	"github.com/choreoatlas2025/cli/internal/input"
	"strings"
	"time"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

type GateResult struct {
	Checked    bool           `json:"checked"`
	Passed     bool           `json:"passed"`
	Details    map[string]any `json:"details"`
	Violations []string       `json:"violations,omitempty"`
}

type InputBinding struct {
	Contract           spec.ContractIdentity   `json:"contract"`
	TraceHash          string                  `json:"traceHash"`
	TraceIdentity      trace.Identity          `json:"traceIdentity"`
	BaselineProvenance *spec.ExecutionIdentity `json:"baselineProvenance,omitempty"`
	BaselineHash       string                  `json:"baselineHash,omitempty"`
	BuildChannel       string                  `json:"buildChannel,omitempty"`
	Version            string                  `json:"version"`
	GitCommit          string                  `json:"gitCommit"`
	Semantic           bool                    `json:"semantic"`
	Causality          string                  `json:"causality"`
	ToleranceMs        int64                   `json:"causalityToleranceMs"`
	ValidatorHash      string                  `json:"validatorHash"`
	Policy             map[string]any          `json:"policy"`
	Limits             input.Limits            `json:"limits"`
	PlanHash           string                  `json:"planHash,omitempty"`
}

// CoverageSummary retains the JSON/JUnit wire fields and their units.
type CoverageSummary struct {
	StepsTotal             int            `json:"stepsTotal"`
	StepsPass              int            `json:"stepsPass"`
	StepsFail              int            `json:"stepsFail"`
	StepsSkip              int            `json:"stepsSkip"`
	ConditionsTotal        int            `json:"conditionsTotal"`
	ConditionsPass         int            `json:"conditionsPass"`
	ConditionsFail         int            `json:"conditionsFail"`
	ConditionsSkip         int            `json:"conditionsSkip"`
	UncoveredSteps         []string       `json:"uncoveredSteps"`
	CoverageRate           float64        `json:"coverageRate"`
	ServiceCoverage        map[string]int `json:"serviceCoverage"`
	BaselineStepsCoverage  float64        `json:"baselineStepsCoverage,omitempty"`
	StepsDeltaAbs          float64        `json:"stepsDeltaAbs,omitempty"`
	StepsDeltaPct          float64        `json:"stepsDeltaPct,omitempty"`
	BaselineConditionsRate float64        `json:"baselineConditionsRate,omitempty"`
	ConditionsDeltaAbs     float64        `json:"conditionsDeltaAbs,omitempty"`
	ConditionsDeltaPct     float64        `json:"conditionsDeltaPct,omitempty"`
}

func (s CoverageSummary) StepsRate() float64 {
	if s.StepsTotal == 0 {
		return 0
	}
	return float64(s.StepsPass) / float64(s.StepsTotal)
}

func (s CoverageSummary) ConditionsRate(skipAsFail bool) float64 {
	total := s.ConditionsPass + s.ConditionsFail
	if skipAsFail {
		total += s.ConditionsSkip
	}
	if total == 0 {
		return 0
	}
	return float64(s.ConditionsPass) / float64(total)
}

// Measure uses the same step success predicate as the final decision. SKIP is
// exposed separately; the legacy uncoveredSteps field lists failed calls.
func Measure(steps []validate.StepResult) CoverageSummary {
	s := CoverageSummary{ServiceCoverage: map[string]int{}, UncoveredSteps: []string{}}
	for _, step := range steps {
		s.StepsTotal++
		switch {
		case validate.StepPassed(step):
			s.StepsPass++
		case step.Status == "SKIP":
			s.StepsSkip++
		default:
			s.StepsFail++
			s.UncoveredSteps = append(s.UncoveredSteps, step.Step)
		}
		if service, _, ok := strings.Cut(step.Call, "."); ok {
			s.ServiceCoverage[service]++
		}
		for _, condition := range step.Conditions {
			s.ConditionsTotal++
			switch condition.Status {
			case "PASS":
				s.ConditionsPass++
			case "SKIP":
				s.ConditionsSkip++
			default:
				s.ConditionsFail++
			}
		}
	}
	s.CoverageRate = s.StepsRate() * 100
	return s
}

func Decision(steps []validate.StepResult, gate *GateResult) validate.Outcome {
	if gate == nil {
		return validate.FinalOutcome(steps, false, true)
	}
	return validate.FinalOutcome(steps, gate.Checked, gate.Passed)
}

func ExitCode(outcome validate.Outcome) int {
	switch outcome.Status {
	case "PASS":
		return 0
	case "VALIDATION_FAILED":
		return 3
	case "GATE_FAILED":
		return 4
	default:
		return 1
	}
}

// Report is the neutral record consumed by the JSON, JUnit and HTML renderers.
type Report struct {
	validate.Outcome
	ExitCode    int                   `json:"exitCode"`
	Inputs      *InputBinding         `json:"inputs,omitempty"`
	Timestamp   time.Time             `json:"timestamp"`
	TotalSteps  int                   `json:"totalSteps"`
	PassedSteps int                   `json:"passedSteps"`
	FailedSteps int                   `json:"failedSteps"`
	Steps       []validate.StepResult `json:"steps"`
	Summary     CoverageSummary       `json:"summary"`
	GateResult  *GateResult           `json:"gateResult,omitempty"`
}

func New(steps []validate.StepResult, gate *GateResult, inputs *InputBinding) Report {
	summary := Measure(steps)
	if gate != nil {
		fields := map[string]*float64{
			"baselineStepsCoverage": &summary.BaselineStepsCoverage, "stepsDeltaAbs": &summary.StepsDeltaAbs, "stepsDeltaPct": &summary.StepsDeltaPct,
			"baselineConditionsRate": &summary.BaselineConditionsRate, "conditionsDeltaAbs": &summary.ConditionsDeltaAbs, "conditionsDeltaPct": &summary.ConditionsDeltaPct,
		}
		for key, dest := range fields {
			if value, ok := gate.Details[key].(float64); ok {
				*dest = value
			}
		}
	}
	outcome := Decision(steps, gate)
	return Report{Outcome: outcome, ExitCode: ExitCode(outcome), Inputs: inputs, Timestamp: time.Now().UTC(), TotalSteps: summary.StepsTotal, PassedSteps: summary.StepsPass, FailedSteps: summary.StepsTotal - summary.StepsPass, Steps: steps, Summary: summary, GateResult: gate}
}
