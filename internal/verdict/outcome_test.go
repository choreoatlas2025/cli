// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package verdict

import "testing"

func TestUnprovenChecksCannotPassByStatusOrThreshold(t *testing.T) {
	for _, issue := range []IssueKind{InvalidContract, MissingEvidence, RuleViolation, StructureMismatch, ExecutionError, "future_issue"} {
		for _, inCondition := range []bool{false, true} {
			step := StepResult{Status: "PASS"}
			if inCondition {
				step.Conditions = []ConditionResult{{Status: "PASS", Issue: issue}}
			} else {
				step.Issue = issue
			}
			if StepPassed(step) || FinalOutcome([]StepResult{step}, true, true).Success {
				t.Fatalf("issue %q condition=%t became success", issue, inCondition)
			}
		}
	}
	if FinalOutcome(nil, true, true).Success {
		t.Fatal("empty evidence became success")
	}
	if !FinalOutcome([]StepResult{{Status: "PASS"}}, false, false).Success {
		t.Fatal("positive control failed")
	}
}
