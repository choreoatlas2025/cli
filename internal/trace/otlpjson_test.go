package trace

import (
	"testing"
)

func TestConvertOTLPToTrace(t *testing.T) {
	// 创建正确的OTLP结构体
	otlpData := OTLPTrace{
		ResourceSpans: []OTLPResourceSpans{
			{
				Resource: OTLPResource{
					Attributes: []OTLPAttribute{
						{
							Key: "service.name",
							Value: OTLPValue{
								StringValue: "testService",
							},
						},
					},
				},
				ScopeSpans: []OTLPScopeSpans{
					{
						Scope: OTLPInstrumentationScope{
							Name: "test-tracer",
						},
						Spans: []OTLPSpan{
							{
								TraceID:           "0102030405060708090a0b0c0d0e0f10",
								SpanID:            "1011121314151617",
								Name:              "testOperation",
								StartTimeUnixNano: "1693910000000000000",
								EndTimeUnixNano:   "1693910000100000000",
								Attributes: []OTLPAttribute{
									{
										Key: "http.status_code",
										Value: OTLPValue{
											IntValue: "200",
										},
									},
								},
								Status: OTLPStatus{
									Code: 1,
								},
							},
						},
					},
				},
			},
		},
	}

	trace, err := convertOTLPToTrace(otlpData)
	if err != nil {
		t.Fatalf("convertOTLPToTrace failed: %v", err)
	}

	// 验证转换结果
	if len(trace.Spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(trace.Spans))
	}

	span := trace.Spans[0]
	if span.Service != "testService" {
		t.Errorf("Expected service 'testService', got '%s'", span.Service)
	}

	if span.Name != "testOperation" {
		t.Errorf("Expected name 'testOperation', got '%s'", span.Name)
	}

	if span.StartNanos != 1693910000000000000 {
		t.Errorf("Expected StartNanos 1693910000000000000, got %d", span.StartNanos)
	}

	// 验证属性转换  
	statusCode, exists := span.Attributes["http.status_code"]
	if !exists {
		t.Error("Expected http.status_code attribute to exist")
	}
	// convertOTLPValue会将intValue转为int64
	if statusCode != int64(200) {
		t.Errorf("Expected http.status_code int64(200), got '%v' (type: %T)", statusCode, statusCode)
	}
}

func TestConvertOTLPValue(t *testing.T) {
	tests := []struct {
		name     string
		input    OTLPValue
		expected any
	}{
		{
			name: "stringValue",
			input: OTLPValue{
				StringValue: "test-string",
			},
			expected: "test-string",
		},
		{
			name: "intValue",
			input: OTLPValue{
				IntValue: "123",
			},
			expected: int64(123),
		},
		{
			name: "boolValue",
			input: OTLPValue{
				BoolValue: true,
			},
			expected: true,
		},
		{
			name: "doubleValue",
			input: OTLPValue{
				DoubleValue: "123.45",
			},
			expected: 123.45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertOTLPValue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParentSpanIdExtraction(t *testing.T) {
	// 测试父子关系的span数据
	otlpData := OTLPTrace{
		ResourceSpans: []OTLPResourceSpans{
			{
				Resource: OTLPResource{
					Attributes: []OTLPAttribute{
						{
							Key: "service.name",
							Value: OTLPValue{
								StringValue: "parentService",
							},
						},
					},
				},
				ScopeSpans: []OTLPScopeSpans{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "0102030405060708090a0b0c0d0e0f10",
								SpanID:            "1011121314151617",
								Name:              "parentOp",
								StartTimeUnixNano: "1693910000000000000",
								EndTimeUnixNano:   "1693910000100000000",
							},
							{
								TraceID:           "0102030405060708090a0b0c0d0e0f10",
								SpanID:            "2021222324252627",
								ParentSpanID:      "1011121314151617",
								Name:              "childOp",
								StartTimeUnixNano: "1693910000050000000",
								EndTimeUnixNano:   "1693910000080000000",
							},
						},
					},
				},
			},
		},
	}

	trace, err := convertOTLPToTrace(otlpData)
	if err != nil {
		t.Fatalf("convertOTLPToTrace failed: %v", err)
	}

	if len(trace.Spans) != 2 {
		t.Fatalf("Expected 2 spans, got %d", len(trace.Spans))
	}

	// 查找子span
	var childSpan *Span
	for i := range trace.Spans {
		if trace.Spans[i].Name == "childOp" {
			childSpan = &trace.Spans[i]
			break
		}
	}

	if childSpan == nil {
		t.Fatal("Child span not found")
	}

	// 验证父span ID设置
	parentSpanId, exists := childSpan.Attributes["otlp.parent_span_id"]
	if !exists {
		t.Error("Expected otlp.parent_span_id attribute to exist")
	}
	if parentSpanId != "1011121314151617" {
		t.Errorf("Expected parent span ID '1011121314151617', got '%v'", parentSpanId)
	}
}

func TestMissingServiceName(t *testing.T) {
	// 测试缺少service.name的情况
	otlpData := OTLPTrace{
		ResourceSpans: []OTLPResourceSpans{
			{
				Resource: OTLPResource{
					Attributes: []OTLPAttribute{},
				},
				ScopeSpans: []OTLPScopeSpans{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "0102030405060708090a0b0c0d0e0f10",
								SpanID:            "1011121314151617",
								Name:              "testOp",
								StartTimeUnixNano: "1693910000000000000",
								EndTimeUnixNano:   "1693910000100000000",
							},
						},
					},
				},
			},
		},
	}

	trace, err := convertOTLPToTrace(otlpData)
	if err != nil {
		t.Fatalf("convertOTLPToTrace failed: %v", err)
	}

	if len(trace.Spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(trace.Spans))
	}

	span := trace.Spans[0]
	if span.Service != "unknown-service" {
		t.Errorf("Expected service 'unknown-service', got '%s'", span.Service)
	}
}

func TestEmptyOTLPData(t *testing.T) {
	// 测试空数据
	otlpData := OTLPTrace{
		ResourceSpans: []OTLPResourceSpans{},
	}

	trace, err := convertOTLPToTrace(otlpData)
	if err != nil {
		t.Fatalf("convertOTLPToTrace failed: %v", err)
	}

	if len(trace.Spans) != 0 {
		t.Errorf("Expected 0 spans, got %d", len(trace.Spans))
	}
}