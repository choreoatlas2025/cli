// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package verdict owns validation records and fail-closed decisions. It has no
// dependency on contracts, traces, evaluators, renderers or the CLI.
package verdict

// IssueKind identifies the responsibility which could not complete a check.
// Every nonempty issue blocks success, including insufficient evidence.
type IssueKind string

const (
	InvalidContract   IssueKind = "invalid_contract"
	MissingEvidence   IssueKind = "missing_evidence"
	RuleViolation     IssueKind = "rule_violation"
	StructureMismatch IssueKind = "structure_mismatch"
	ExecutionError    IssueKind = "execution_error"
)

// SpanRef refers to the exact matched instance. Key is an explicit span ID or
// the legacy service/name/start key; file identity is carried by report inputs.
type SpanRef struct {
	Key     string `json:"key"`
	TraceID string `json:"traceId,omitempty"`
	SpanID  string `json:"spanId,omitempty"`
}

// FieldSource binds a CEL observation to its captured attribute. Declarations
// and computed expectations never become observation sources.
type FieldSource struct {
	Variable  string  `json:"variable"`
	Attribute string  `json:"attribute"`
	Span      SpanRef `json:"span"`
}

// StepResult 表示单个步骤的验证结果
type StepResult struct {
	Step       string            `json:"step"`
	Call       string            `json:"call"`
	Status     string            `json:"status"` // PASS / FAIL
	Issue      IssueKind         `json:"issue,omitempty"`
	Evidence   *SpanRef          `json:"evidence,omitempty"`
	Bindings   []FieldSource     `json:"bindings,omitempty"`
	Message    string            `json:"message,omitempty"`
	Conditions []ConditionResult `json:"conditions,omitempty"`
	Violations []EdgeViolation   `json:"violations,omitempty"`
}

// 条件结果
type ConditionResult struct {
	Kind    string    `json:"kind"` // "pre" | "post"
	Name    string    `json:"name"`
	Expr    string    `json:"expr"`
	Status  string    `json:"status"` // "PASS" | "FAIL" | "SKIP"
	Issue   IssueKind `json:"issue,omitempty"`
	Message string    `json:"message"` // 失败/跳过原因
}

// EdgeViolation 表示边约束违规
type EdgeViolation struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Type    string `json:"type"` // "cycle", "causality", "overlap"
	Message string `json:"message"`
}
