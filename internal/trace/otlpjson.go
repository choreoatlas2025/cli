package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// OTLPSpan OTLP 格式的 span 定义
type OTLPSpan struct {
	TraceID           string                 `json:"traceId"`
	SpanID            string                 `json:"spanId"`
	ParentSpanID      string                 `json:"parentSpanId,omitempty"`
	Name              string                 `json:"name"`
	StartTimeUnixNano string                 `json:"startTimeUnixNano"`
	EndTimeUnixNano   string                 `json:"endTimeUnixNano"`
	Attributes        []OTLPAttribute        `json:"attributes,omitempty"`
	Events            []OTLPEvent            `json:"events,omitempty"`
	Status            OTLPStatus             `json:"status,omitempty"`
}

// OTLPAttribute OTLP 属性定义
type OTLPAttribute struct {
	Key   string    `json:"key"`
	Value OTLPValue `json:"value"`
}

// OTLPValue OTLP 值定义
type OTLPValue struct {
	StringValue  string `json:"stringValue,omitempty"`
	IntValue     string `json:"intValue,omitempty"`
	DoubleValue  string `json:"doubleValue,omitempty"`
	BoolValue    bool   `json:"boolValue,omitempty"`
	ArrayValue   any    `json:"arrayValue,omitempty"`
	KvlistValue  any    `json:"kvlistValue,omitempty"`
	BytesValue   string `json:"bytesValue,omitempty"`
}

// OTLPEvent OTLP 事件定义
type OTLPEvent struct {
	TimeUnixNano string          `json:"timeUnixNano"`
	Name         string          `json:"name"`
	Attributes   []OTLPAttribute `json:"attributes,omitempty"`
}

// OTLPStatus OTLP 状态定义
type OTLPStatus struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// OTLPResource OTLP 资源定义
type OTLPResource struct {
	Attributes []OTLPAttribute `json:"attributes,omitempty"`
}

// OTLPInstrumentationScope OTLP 仪表化作用域定义
type OTLPInstrumentationScope struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// OTLPScopeSpans OTLP 作用域 spans 定义
type OTLPScopeSpans struct {
	Scope OTLPInstrumentationScope `json:"scope"`
	Spans []OTLPSpan               `json:"spans"`
}

// OTLPResourceSpans OTLP 资源 spans 定义
type OTLPResourceSpans struct {
	Resource   OTLPResource     `json:"resource"`
	ScopeSpans []OTLPScopeSpans `json:"scopeSpans"`
}

// OTLPTrace OTLP 追踪数据根结构
type OTLPTrace struct {
	ResourceSpans []OTLPResourceSpans `json:"resourceSpans"`
}

// LoadFromOTLPJSON 从 OTLP JSON 文件加载追踪数据
func LoadFromOTLPJSON(path string) (*Trace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 OTLP JSON 文件失败: %w", err)
	}

	var otlpTrace OTLPTrace
	if err := json.Unmarshal(data, &otlpTrace); err != nil {
		return nil, fmt.Errorf("解析 OTLP JSON 失败: %w", err)
	}

	return convertOTLPToTrace(otlpTrace)
}

// convertOTLPToTrace 将 OTLP 格式转换为内部 Trace 格式
func convertOTLPToTrace(otlpTrace OTLPTrace) (*Trace, error) {
	var spans []Span

	for _, resourceSpan := range otlpTrace.ResourceSpans {
		// 提取服务名（优先从 resource.attributes 中获取 service.name）
		serviceName := extractServiceName(resourceSpan.Resource)

		for _, scopeSpan := range resourceSpan.ScopeSpans {
			for _, otlpSpan := range scopeSpan.Spans {
				span, err := convertOTLPSpan(otlpSpan, serviceName)
				if err != nil {
					return nil, fmt.Errorf("转换 OTLP span 失败: %w", err)
				}
				spans = append(spans, span)
			}
		}
	}

	return &Trace{Spans: spans}, nil
}

// convertOTLPSpan 转换单个 OTLP span
func convertOTLPSpan(otlpSpan OTLPSpan, defaultService string) (Span, error) {
	// 转换时间戳
	startNanos, err := strconv.ParseInt(otlpSpan.StartTimeUnixNano, 10, 64)
	if err != nil {
		return Span{}, fmt.Errorf("解析开始时间失败: %w", err)
	}

	endNanos, err := strconv.ParseInt(otlpSpan.EndTimeUnixNano, 10, 64)
	if err != nil {
		return Span{}, fmt.Errorf("解析结束时间失败: %w", err)
	}

	// 转换属性
	attributes := make(map[string]any)
	for _, attr := range otlpSpan.Attributes {
		value := convertOTLPValue(attr.Value)
		attributes[attr.Key] = value
	}

	// 添加 OTLP 特有的元数据
	attributes["otlp.trace_id"] = otlpSpan.TraceID
	attributes["otlp.span_id"] = otlpSpan.SpanID
	if otlpSpan.ParentSpanID != "" {
		attributes["otlp.parent_span_id"] = otlpSpan.ParentSpanID
	}

	// 添加状态信息
	if otlpSpan.Status.Code != 0 {
		attributes["otlp.status.code"] = otlpSpan.Status.Code
	}
	if otlpSpan.Status.Message != "" {
		attributes["otlp.status.message"] = otlpSpan.Status.Message
	}

	// 处理 HTTP 状态码映射
	if httpStatusCode, exists := attributes["http.status_code"]; exists {
		attributes["response.status"] = httpStatusCode
	} else if otlpSpan.Status.Code == 1 { // OTLP OK status
		// 根据操作类型推断默认状态码
		if isCreationOperation(otlpSpan.Name) {
			attributes["response.status"] = 201
		} else {
			attributes["response.status"] = 200
		}
	}

	// 确定服务名
	serviceName := defaultService
	if svcName, exists := attributes["service.name"]; exists {
		if svcStr, ok := svcName.(string); ok {
			serviceName = svcStr
		}
	}
	if serviceName == "" {
		serviceName = "unknown-service"
	}

	return Span{
		Name:       otlpSpan.Name,
		Service:    serviceName,
		StartNanos: startNanos,
		EndNanos:   endNanos,
		Attributes: attributes,
	}, nil
}

// convertOTLPValue 转换 OTLP 值为 Go 原生类型
func convertOTLPValue(value OTLPValue) any {
	if value.StringValue != "" {
		return value.StringValue
	}
	if value.IntValue != "" {
		if intVal, err := strconv.ParseInt(value.IntValue, 10, 64); err == nil {
			return intVal
		}
		return value.IntValue // 保持字符串形式
	}
	if value.DoubleValue != "" {
		if floatVal, err := strconv.ParseFloat(value.DoubleValue, 64); err == nil {
			return floatVal
		}
		return value.DoubleValue // 保持字符串形式
	}
	if value.BoolValue {
		return value.BoolValue
	}
	if value.ArrayValue != nil {
		return value.ArrayValue
	}
	if value.KvlistValue != nil {
		return value.KvlistValue
	}
	if value.BytesValue != "" {
		return value.BytesValue
	}
	return ""
}

// extractServiceName 从资源属性中提取服务名
func extractServiceName(resource OTLPResource) string {
	for _, attr := range resource.Attributes {
		if attr.Key == "service.name" {
			if serviceName := convertOTLPValue(attr.Value); serviceName != nil {
				if str, ok := serviceName.(string); ok {
					return str
				}
			}
		}
	}
	return ""
}

// isCreationOperation 判断是否为创建操作
func isCreationOperation(operationName string) bool {
	lowerName := strings.ToLower(operationName)
	return strings.Contains(lowerName, "create") ||
		strings.Contains(lowerName, "post") ||
		strings.Contains(lowerName, "insert") ||
		strings.Contains(lowerName, "add")
}

// ValidateOTLPJSON 验证 OTLP JSON 文件格式
func ValidateOTLPJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var otlpTrace OTLPTrace
	if err := json.Unmarshal(data, &otlpTrace); err != nil {
		return fmt.Errorf("JSON 格式无效: %w", err)
	}

	if len(otlpTrace.ResourceSpans) == 0 {
		return fmt.Errorf("OTLP 文件中没有找到 resourceSpans")
	}

	totalSpans := 0
	for _, rs := range otlpTrace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			totalSpans += len(ss.Spans)
		}
	}

	if totalSpans == 0 {
		return fmt.Errorf("OTLP 文件中没有找到任何 spans")
	}

	return nil
}

// GetOTLPStats 获取 OTLP 文件统计信息
func GetOTLPStats(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var otlpTrace OTLPTrace
	if err := json.Unmarshal(data, &otlpTrace); err != nil {
		return nil, fmt.Errorf("JSON 格式无效: %w", err)
	}

	stats := map[string]any{
		"resourceSpans": len(otlpTrace.ResourceSpans),
		"totalSpans":    0,
		"services":      make(map[string]int),
		"operations":    make(map[string]int),
	}

	serviceStats := make(map[string]int)
	operationStats := make(map[string]int)
	totalSpans := 0

	for _, rs := range otlpTrace.ResourceSpans {
		serviceName := extractServiceName(rs.Resource)
		if serviceName == "" {
			serviceName = "unknown-service"
		}

		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				totalSpans++
				serviceStats[serviceName]++
				operationStats[span.Name]++
			}
		}
	}

	stats["totalSpans"] = totalSpans
	stats["services"] = serviceStats
	stats["operations"] = operationStats

	return stats, nil
}