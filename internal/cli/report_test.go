package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/choreoatlas2025/cli/internal/validate"
)

func TestCalculateCoverageSummary(t *testing.T) {
	// 创建测试步骤结果
	steps := []validate.StepResult{
		{
			Step:   "成功步骤",
			Call:   "serviceA.operation1",
			Status: "PASS",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "条件1", Status: "PASS"},
				{Kind: "post", Name: "条件2", Status: "PASS"},
			},
		},
		{
			Step:   "失败步骤",
			Call:   "serviceB.operation2", 
			Status: "FAIL",
			Conditions: []validate.ConditionResult{
				{Kind: "pre", Name: "条件3", Status: "FAIL"},
				{Kind: "post", Name: "条件4", Status: "SKIP"},
			},
		},
		{
			Step:   "跳过步骤",
			Call:   "serviceA.operation3",
			Status: "SKIP",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "条件5", Status: "SKIP"},
			},
		},
	}

	summary := calculateCoverageSummary(steps)

	// 验证步骤统计
	if summary.StepsTotal != 3 {
		t.Errorf("Expected StepsTotal 3, got %d", summary.StepsTotal)
	}
	if summary.StepsPass != 1 {
		t.Errorf("Expected StepsPass 1, got %d", summary.StepsPass)
	}
	if summary.StepsFail != 1 {
		t.Errorf("Expected StepsFail 1, got %d", summary.StepsFail)
	}
	if summary.StepsSkip != 1 {
		t.Errorf("Expected StepsSkip 1, got %d", summary.StepsSkip)
	}

	// 验证条件统计
	if summary.ConditionsTotal != 5 {
		t.Errorf("Expected ConditionsTotal 5, got %d", summary.ConditionsTotal)
	}
	if summary.ConditionsPass != 2 {
		t.Errorf("Expected ConditionsPass 2, got %d", summary.ConditionsPass)
	}
	if summary.ConditionsFail != 1 {
		t.Errorf("Expected ConditionsFail 1, got %d", summary.ConditionsFail)
	}
	if summary.ConditionsSkip != 2 {
		t.Errorf("Expected ConditionsSkip 2, got %d", summary.ConditionsSkip)
	}

	// 验证覆盖率计算
	expectedRate := float64(1) / float64(3) * 100
	if summary.CoverageRate != expectedRate {
		t.Errorf("Expected CoverageRate %.2f, got %.2f", expectedRate, summary.CoverageRate)
	}

	// 验证未覆盖步骤
	if len(summary.UncoveredSteps) != 1 {
		t.Errorf("Expected 1 uncovered step, got %d", len(summary.UncoveredSteps))
	}
	if summary.UncoveredSteps[0] != "失败步骤" {
		t.Errorf("Expected uncovered step '失败步骤', got '%s'", summary.UncoveredSteps[0])
	}

	// 验证服务覆盖度
	if len(summary.ServiceCoverage) != 2 {
		t.Errorf("Expected 2 services, got %d", len(summary.ServiceCoverage))
	}
	if summary.ServiceCoverage["serviceA"] != 2 {
		t.Errorf("Expected serviceA coverage 2, got %d", summary.ServiceCoverage["serviceA"])
	}
	if summary.ServiceCoverage["serviceB"] != 1 {
		t.Errorf("Expected serviceB coverage 1, got %d", summary.ServiceCoverage["serviceB"])
	}
}

func TestWriteJSONReport(t *testing.T) {
	steps := []validate.StepResult{
		{
			Step:   "测试步骤",
			Call:   "testService.testOp",
			Status: "PASS",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "测试条件", Status: "PASS"},
			},
		},
	}

	tempFile := "/tmp/test-report.json"
	err := writeJSONReport(tempFile, steps)
	if err != nil {
		t.Fatalf("writeJSONReport failed: %v", err)
	}
	defer os.Remove(tempFile)

	// 读取并验证生成的JSON
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	var report struct {
		Timestamp   time.Time                `json:"timestamp"`
		TotalSteps  int                      `json:"totalSteps"`
		PassedSteps int                      `json:"passedSteps"`
		FailedSteps int                      `json:"failedSteps"`
		Success     bool                     `json:"success"`
		Steps       []validate.StepResult    `json:"steps"`
		Summary     CoverageSummary          `json:"summary"`
	}

	err = json.Unmarshal(data, &report)
	if err != nil {
		t.Fatalf("Failed to parse JSON report: %v", err)
	}

	// 验证报告内容
	if report.TotalSteps != 1 {
		t.Errorf("Expected TotalSteps 1, got %d", report.TotalSteps)
	}
	if report.PassedSteps != 1 {
		t.Errorf("Expected PassedSteps 1, got %d", report.PassedSteps)
	}
	if report.FailedSteps != 0 {
		t.Errorf("Expected FailedSteps 0, got %d", report.FailedSteps)
	}
	if !report.Success {
		t.Error("Expected Success true, got false")
	}

	// 验证包含了覆盖度摘要
	if report.Summary.StepsTotal != 1 {
		t.Errorf("Expected Summary.StepsTotal 1, got %d", report.Summary.StepsTotal)
	}
	if report.Summary.CoverageRate != 100.0 {
		t.Errorf("Expected Summary.CoverageRate 100.0, got %.2f", report.Summary.CoverageRate)
	}
}

func TestWriteJUnitReport(t *testing.T) {
	steps := []validate.StepResult{
		{
			Step:   "JUnit测试步骤",
			Call:   "junitService.junitOp",
			Status: "PASS",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "JUnit条件", Status: "PASS"},
			},
		},
		{
			Step:    "失败步骤",
			Call:    "failService.failOp", 
			Status:  "FAIL",
			Message: "测试失败消息",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "失败条件", Status: "FAIL"},
			},
		},
	}

	tempFile := "/tmp/test-junit.xml"
	err := writeJUnitReport(tempFile, steps)
	if err != nil {
		t.Fatalf("writeJUnitReport failed: %v", err)
	}
	defer os.Remove(tempFile)

	// 读取并验证生成的XML
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read JUnit file: %v", err)
	}

	content := string(data)

	// 验证XML结构和内容
	if !strings.Contains(content, `<testsuite name="flowspec-validation" tests="2" failures="1"`) {
		t.Error("JUnit XML should contain correct testsuite header")
	}

	// 验证覆盖度属性
	if !strings.Contains(content, `<property name="coverage.stepsTotal" value="2"/>`) {
		t.Error("JUnit XML should contain coverage.stepsTotal property")
	}
	if !strings.Contains(content, `<property name="coverage.stepsPass" value="1"/>`) {
		t.Error("JUnit XML should contain coverage.stepsPass property")
	}
	if !strings.Contains(content, `<property name="coverage.stepsFail" value="1"/>`) {
		t.Error("JUnit XML should contain coverage.stepsFail property")
	}

	// 验证测试用例
	if !strings.Contains(content, `<testcase name="JUnit测试步骤" classname="junitService.junitOp">`) {
		t.Error("JUnit XML should contain passing test case")
	}
	if !strings.Contains(content, `<testcase name="失败步骤" classname="failService.failOp">`) {
		t.Error("JUnit XML should contain failing test case")
	}

	// 验证失败信息
	if !strings.Contains(content, `<failure message="测试失败消息"`) {
		t.Error("JUnit XML should contain failure message")
	}

	// 验证条件详情在system-out中
	if !strings.Contains(content, `<system-out><![CDATA[`) {
		t.Error("JUnit XML should contain conditions in system-out")
	}

	// 验证覆盖度摘要在最终的system-out中  
	if !strings.Contains(content, `"stepsTotal": 2`) {
		t.Error("JUnit XML should contain coverage summary in system-out")
	}
}

func TestXmlEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text with & ampersand", "text with &amp; ampersand"},
		{"<tag>content</tag>", "&lt;tag&gt;content&lt;/tag&gt;"},
		{`"quoted" and 'single'`, "&quot;quoted&quot; and &apos;single&apos;"},
		{"mixed <>&\"'", "mixed &lt;&gt;&amp;&quot;&apos;"},
	}

	for _, tt := range tests {
		result := xmlEscape(tt.input)
		if result != tt.expected {
			t.Errorf("xmlEscape(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestWriteReportFormats(t *testing.T) {
	steps := []validate.StepResult{
		{
			Step:   "格式测试",
			Call:   "formatService.formatOp",
			Status: "PASS",
		},
	}

	// 测试JSON格式
	jsonFile := "/tmp/test-format.json"
	err := WriteReport(jsonFile, ReportJSON, steps, nil, nil)
	if err != nil {
		t.Errorf("WriteReport JSON failed: %v", err)
	} else {
		os.Remove(jsonFile)
	}

	// 测试JUnit格式
	xmlFile := "/tmp/test-format.xml"
	err = WriteReport(xmlFile, ReportJUnit, steps, nil, nil)
	if err != nil {
		t.Errorf("WriteReport JUnit failed: %v", err)
	} else {
		os.Remove(xmlFile)
	}

	// 测试不支持的格式
	err = WriteReport("/tmp/test-unknown.txt", "unknown", steps, nil, nil)
	if err == nil {
		t.Error("WriteReport should fail for unknown format")
	}
	if !strings.Contains(err.Error(), "不支持的报告格式") {
		t.Errorf("Expected format error message, got: %v", err)
	}
}

func TestEmptyStepsReport(t *testing.T) {
	// 测试空步骤列表
	var steps []validate.StepResult

	summary := calculateCoverageSummary(steps)
	
	if summary.StepsTotal != 0 {
		t.Errorf("Expected StepsTotal 0 for empty steps, got %d", summary.StepsTotal)
	}
	if summary.CoverageRate != 0 {
		t.Errorf("Expected CoverageRate 0 for empty steps, got %.2f", summary.CoverageRate)
	}
	if len(summary.UncoveredSteps) != 0 {
		t.Errorf("Expected no uncovered steps for empty list, got %d", len(summary.UncoveredSteps))
	}
}

func TestServiceCoverageExtraction(t *testing.T) {
	steps := []validate.StepResult{
		{Call: "service1.op1", Status: "PASS"},
		{Call: "service1.op2", Status: "PASS"},
		{Call: "service2.op1", Status: "FAIL"},
		{Call: "invalid.call.format", Status: "PASS"}, // 应该被忽略
		{Call: "", Status: "PASS"}, // 应该被忽略
	}

	summary := calculateCoverageSummary(steps)

	expectedServices := map[string]int{
		"service1": 2,
		"service2": 1,
		"invalid": 1, // invalid.call.format被解析为invalid服务
	}

	if len(summary.ServiceCoverage) != len(expectedServices) {
		t.Errorf("Expected %d services, got %d", len(expectedServices), len(summary.ServiceCoverage))
	}

	for service, expectedCount := range expectedServices {
		if actualCount, exists := summary.ServiceCoverage[service]; !exists {
			t.Errorf("Expected service %s not found", service)
		} else if actualCount != expectedCount {
			t.Errorf("Expected service %s count %d, got %d", service, expectedCount, actualCount)
		}
	}
}