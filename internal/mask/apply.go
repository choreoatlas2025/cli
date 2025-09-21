// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package mask

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/choreoatlas2025/cli/internal/trace"
)

// Apply 对 trace 应用脱敏策略
func Apply(policy *CompiledPolicy, tr *trace.Trace) (*trace.Trace, error) {
	// 深拷贝 trace 以避免修改原数据
	maskedTrace := &trace.Trace{
		Spans: make([]trace.Span, len(tr.Spans)),
	}

	for i, span := range tr.Spans {
		maskedSpan := trace.Span{
			Name:       span.Name,
			Service:    span.Service,
			StartNanos: span.StartNanos,
			EndNanos:   span.EndNanos,
			Attributes: make(map[string]any),
		}

		// 拷贝并脱敏 attributes
		for key, value := range span.Attributes {
			maskedSpan.Attributes[key] = applyMaskingRules(policy, span.Service, span.Name, key, value, []string{key})
		}

		maskedTrace.Spans[i] = maskedSpan
	}

	return maskedTrace, nil
}

// applyMaskingRules 对单个值应用脱敏规则
func applyMaskingRules(policy *CompiledPolicy, service, operation, key string, value any, path []string) any {
	// 查找匹配的规则
	for _, rule := range policy.Rules {
		if matchesSelector(rule.Selector, service, operation, key, path) {
			return applyMaskingToValue(value, rule.Strategy, path)
		}
	}
	
	// 如果是复合对象，递归处理
	return applyMaskingToValue(value, Strategy{}, path)
}

// matchesSelector 检查是否匹配选择器条件
func matchesSelector(selector CompiledSelector, service, operation, key string, path []string) bool {
	// 检查 service 匹配
	if selector.Service != "" && selector.Service != service {
		return false
	}

	// 检查 operation 匹配
	if selector.Operation != "" && selector.Operation != operation {
		return false
	}

	// 检查路径匹配
	if len(selector.Paths) > 0 {
		pathStr := strings.Join(path, ".")
		matched := false
		for _, selectorPath := range selector.Paths {
			if pathStr == selectorPath || strings.HasSuffix(pathStr, "."+selectorPath) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查正则匹配
	if len(selector.RegexMatchers) > 0 {
		matched := false
		for _, regex := range selector.RegexMatchers {
			if regex.MatchString(key) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// TODO: 检查标签匹配（预留，等 ServiceSpec 支持标签）
	// if selector.Tag != "" { ... }

	return true
}

// applyMaskingToValue 对值应用脱敏策略
func applyMaskingToValue(value any, strategy Strategy, path []string) any {
	if strategy.Type != "" {
		// 有明确的脱敏策略
		return ApplyStrategy(value, strategy)
	}

	// 没有明确策略，递归处理复合对象
	switch v := value.(type) {
	case map[string]any:
		masked := make(map[string]any)
		for key, val := range v {
			newPath := append(path, key)
			masked[key] = applyMaskingToValue(val, Strategy{}, newPath)
		}
		return masked
	case []any:
		masked := make([]any, len(v))
		for i, val := range v {
			indexPath := append(path, fmt.Sprintf("[%d]", i))
			masked[i] = applyMaskingToValue(val, Strategy{}, indexPath)
		}
		return masked
	default:
		return value
	}
}

// ApplyToSpan 对单个 span 应用脱敏策略
func ApplyToSpan(policy *CompiledPolicy, span trace.Span) trace.Span {
	maskedSpan := trace.Span{
		Name:       span.Name,
		Service:    span.Service,
		StartNanos: span.StartNanos,
		EndNanos:   span.EndNanos,
		Attributes: make(map[string]any),
	}

	for key, value := range span.Attributes {
		maskedSpan.Attributes[key] = applyMaskingRules(policy, span.Service, span.Name, key, value, []string{key})
	}

	return maskedSpan
}

// MaskJSON 对 JSON 数据应用脱敏策略（用于测试和调试）
func MaskJSON(policy *CompiledPolicy, service, operation string, data map[string]any) map[string]any {
	masked := make(map[string]any)
	
	for key, value := range data {
		masked[key] = applyMaskingRules(policy, service, operation, key, value, []string{key})
	}
	
	return masked
}

// ValidateAndApply 验证策略并应用脱敏
func ValidateAndApply(policyPath string, tr *trace.Trace) (*trace.Trace, error) {
	// 加载策略
	policy, err := LoadPolicy(policyPath)
	if err != nil {
		return nil, fmt.Errorf("加载脱敏策略失败: %w", err)
	}

	// 编译策略
	compiled, err := policy.Compile()
	if err != nil {
		return nil, fmt.Errorf("编译脱敏策略失败: %w", err)
	}

	// 应用脱敏
	return Apply(compiled, tr)
}

// ShowMaskingPreview displays masking preview (for debugging)
func ShowMaskingPreview(policy *CompiledPolicy, tr *trace.Trace, maxSpans int) {
	fmt.Printf("Masking preview (showing first %d spans):\n", maxSpans)
	fmt.Println(strings.Repeat("=", 60))

	count := 0
	for _, span := range tr.Spans {
		if count >= maxSpans {
			break
		}

		fmt.Printf("\nSpan: %s.%s\n", span.Service, span.Name)
		fmt.Println("Original data:")
		
		// 格式化输出原始 attributes
		originalJSON, _ := json.MarshalIndent(span.Attributes, "  ", "  ")
		fmt.Printf("  %s\n", originalJSON)

		// 应用脱敏
		maskedSpan := ApplyToSpan(policy, span)
		fmt.Println("After masking:")
		
		maskedJSON, _ := json.MarshalIndent(maskedSpan.Attributes, "  ", "  ")
		fmt.Printf("  %s\n", maskedJSON)

		count++
	}

	if len(tr.Spans) > maxSpans {
		fmt.Printf("\n... %d more spans not shown\n", len(tr.Spans)-maxSpans)
	}
}

// GetMaskingStats 获取脱敏统计信息
func GetMaskingStats(policy *CompiledPolicy, tr *trace.Trace) MaskingStats {
	stats := MaskingStats{
		TotalSpans:      len(tr.Spans),
		TotalAttributes: 0,
		MaskedAttributes: 0,
		RulesApplied:    make(map[string]int),
	}

	for _, span := range tr.Spans {
		for key := range span.Attributes {
			stats.TotalAttributes++
			
			// 检查是否被脱敏
			for _, rule := range policy.Rules {
				if matchesSelector(rule.Selector, span.Service, span.Name, key, []string{key}) {
					stats.MaskedAttributes++
					
					// 统计规则应用次数
					ruleKey := fmt.Sprintf("%s-%s", rule.Selector.Service, rule.Strategy.Type)
					if rule.Selector.Service == "" {
						ruleKey = fmt.Sprintf("global-%s", rule.Strategy.Type)
					}
					stats.RulesApplied[ruleKey]++
					break
				}
			}
		}
	}

	return stats
}

// MaskingStats 脱敏统计信息
type MaskingStats struct {
	TotalSpans       int            `json:"totalSpans"`
	TotalAttributes  int            `json:"totalAttributes"`
	MaskedAttributes int            `json:"maskedAttributes"`
	RulesApplied     map[string]int `json:"rulesApplied"`
}

// DeepCopy 深拷贝任意值
func DeepCopy(src any) any {
	if src == nil {
		return nil
	}

	// 使用反射进行深拷贝
	original := reflect.ValueOf(src)
	copy := reflect.New(original.Type()).Elem()
	copyRecursive(original, copy)
	return copy.Interface()
}

// copyRecursive 递归拷贝
func copyRecursive(original, copy reflect.Value) {
	switch original.Kind() {
	case reflect.Ptr:
		if !original.IsNil() {
			copy.Set(reflect.New(original.Elem().Type()))
			copyRecursive(original.Elem(), copy.Elem())
		}
	case reflect.Interface:
		if !original.IsNil() {
			copyRecursive(original.Elem(), copy)
		}
	case reflect.Struct:
		for i := 0; i < original.NumField(); i++ {
			if copy.Field(i).CanSet() {
				copyRecursive(original.Field(i), copy.Field(i))
			}
		}
	case reflect.Slice:
		if !original.IsNil() {
			copy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
			for i := 0; i < original.Len(); i++ {
				copyRecursive(original.Index(i), copy.Index(i))
			}
		}
	case reflect.Map:
		if !original.IsNil() {
			copy.Set(reflect.MakeMap(original.Type()))
			for _, key := range original.MapKeys() {
				originalValue := original.MapIndex(key)
				copyValue := reflect.New(originalValue.Type()).Elem()
				copyRecursive(originalValue, copyValue)
				copy.SetMapIndex(key, copyValue)
			}
		}
	default:
		copy.Set(original)
	}
}