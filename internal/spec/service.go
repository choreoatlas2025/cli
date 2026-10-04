// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/choreoatlas2025/cli/internal/trace"
	"gopkg.in/yaml.v3"
)

// ServiceSpecFile 表示服务规约文件（文件内可包含多个 operation）
type ServiceSpecFile struct {
	Service    string             `yaml:"service"`
	Operations []ServiceOperation `yaml:"operations"`
}

// ServiceOperation 表示服务的一个操作
type ServiceOperation struct {
	OperationId    string            `yaml:"operationId"`
	Description    string            `yaml:"description,omitempty"`
	Preconditions  map[string]string `yaml:"preconditions,omitempty"`  // 可 CEL 表达式（预留）
	Postconditions map[string]string `yaml:"postconditions,omitempty"` // 可 CEL 表达式（预留）
}

// LoadServiceSpec 从文件加载服务规约
func LoadServiceSpec(path string) (*ServiceSpecFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read servicespec: %w", err)
	}
	var ss ServiceSpecFile
	if err := yaml.Unmarshal(b, &ss); err != nil {
		return nil, fmt.Errorf("failed to parse servicespec: %w", err)
	}
	return &ss, nil
}

// BuildServiceSpecFiles generates data without changing any destination files.
func BuildServiceSpecFiles(spans []trace.Span) (map[string][]byte, error) {
	files := map[string][]byte{}
	for service, operations := range groupSpansByService(spans) {
		name := ServiceSpecFilename(service)
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("invalid service names: filename collision for %s", name)
		}
		sort.Slice(operations, func(i, j int) bool { return operations[i].OperationId < operations[j].OperationId })
		data, err := yaml.Marshal(&ServiceSpecFile{Service: service, Operations: operations})
		if err != nil {
			return nil, err
		}
		files[name] = data
	}
	return files, nil
}

func ServiceSpecFilename(service string) string {
	return normalizeServiceName(service) + ".servicespec.yaml"
}

func GenerateServiceSpecs(spans []trace.Span, outDir string) error {
	files, err := BuildServiceSpecFiles(spans)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(outDir, name)
		if err := os.WriteFile(path, files[name], 0o644); err != nil {
			return err
		}
		fmt.Printf("Generated ServiceSpec: %s\n", path)
	}
	return nil
}

// groupSpansByService 按服务分组 spans 并生成操作
func groupSpansByService(spans []trace.Span) map[string][]ServiceOperation {
	serviceOps := make(map[string][]ServiceOperation)

	// 按服务和操作分组
	opGroups := make(map[string]map[string][]trace.Span)

	for _, span := range spans {
		if span.Service == "" || span.Name == "" {
			continue
		}

		service := span.Service
		opName := ComputeOperationID(span)

		if _, exists := opGroups[service]; !exists {
			opGroups[service] = make(map[string][]trace.Span)
		}
		opGroups[service][opName] = append(opGroups[service][opName], span)
	}

	// 为每个服务的每个操作生成 ServiceOperation
	for service, ops := range opGroups {
		var operations []ServiceOperation
		used := map[string]bool{}
		for opName, spanList := range ops {
			// Simple collision handling: append _2, _3...
			base := opName
			c := 2
			for used[opName] {
				opName = fmt.Sprintf("%s_%d", base, c)
				c++
			}
			used[opName] = true
			op := generateServiceOperation(opName, spanList)
			operations = append(operations, op)
		}
		serviceOps[service] = operations
	}

	return serviceOps
}

// generateServiceOperation 从 span 列表生成单个 ServiceOperation
func generateServiceOperation(opName string, spans []trace.Span) ServiceOperation {
	preconditions := make(map[string]string)
	postconditions := make(map[string]string)

	// 从所有相关 spans 的 attributes 中提取条件
	for _, span := range spans {
		for key, value := range span.Attributes {
			if celExpr := buildCELExpression(key, value); celExpr != "" {
				if isRequestAttribute(key) {
					// 请求相关属性生成前置条件
					conditionName := generateConditionName("req", key)
					preconditions[conditionName] = celExpr
				} else if isResponseAttribute(key) {
					// 响应相关属性生成后置条件
					conditionName := generateConditionName("resp", key)
					postconditions[conditionName] = celExpr
				}
			}
		}
	}

	return ServiceOperation{
		OperationId:    opName,
		Description:    fmt.Sprintf("Auto-generated %s operation from trace", opName),
		Preconditions:  preconditions,
		Postconditions: postconditions,
	}
}

// buildCELExpression 根据属性键值生成 CEL 表达式
func buildCELExpression(key string, value interface{}) string {
	keyLower := strings.ToLower(key)

	// Special cases for common HTTP request attributes
	if keyLower == "http.method" {
		if s, ok := value.(string); ok && s != "" {
			return fmt.Sprintf("http.method == '%s'", s)
		}
	}
	if keyLower == "http.route" || keyLower == "http.target" {
		if s, ok := value.(string); ok && s != "" {
			return fmt.Sprintf("http.route == '%s'", s)
		}
	}

	// 状态码检查
	if isStatusAttribute(key, value) {
		switch v := value.(type) {
		case int:
			return fmt.Sprintf("response.status == %d", v)
		case int64:
			return fmt.Sprintf("response.status == %d", int(v))
		case float64:
			return fmt.Sprintf("response.status == %d", int(v))
		}
	}

	// Bearer token 检查
	if isBearerToken(key, value) {
		return "request.headers.authorization =~ /Bearer .+/"
	}

	// 字符串非空检查
	if strVal, ok := value.(string); ok && strVal != "" {
		if strings.Contains(keyLower, "request") || strings.Contains(keyLower, "body") {
			return fmt.Sprintf("request.body.%s != \"\"", extractFieldName(key))
		} else if strings.Contains(keyLower, "response") {
			return fmt.Sprintf("response.body.%s != \"\"", extractFieldName(key))
		}
	}

	return ""
}

// 辅助函数
func isStatusAttribute(key string, value interface{}) bool {
	keyLower := strings.ToLower(key)
	statusPatterns := []string{"status", "statuscode", "http.status_code", "response.status"}

	for _, pattern := range statusPatterns {
		if strings.Contains(keyLower, pattern) {
			// 检查值是否为数字类型
			switch value.(type) {
			case int, int64, float64:
				return true
			}
		}
	}
	return false
}

func isBearerToken(key string, value interface{}) bool {
	keyLower := strings.ToLower(key)
	if strings.Contains(keyLower, "authorization") {
		if strVal, ok := value.(string); ok {
			return strings.Contains(strings.ToLower(strVal), "bearer")
		}
	}
	return false
}

func isRequestAttribute(key string) bool {
	keyLower := strings.ToLower(key)
	requestPatterns := []string{"request.", "http.method", "http.url", "http.route", "http.target"}

	for _, pattern := range requestPatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}
	return false
}

func isResponseAttribute(key string) bool {
	keyLower := strings.ToLower(key)
	responsePatterns := []string{"response.", "http.status_code"}

	for _, pattern := range responsePatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}
	return false
}

func extractFieldName(key string) string {
	// 提取字段名，例如 "request.body.username" -> "username"
	parts := strings.Split(key, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return key
}

func generateConditionName(prefix, key string) string {
	fieldName := extractFieldName(key)
	return fmt.Sprintf("%s_%s", prefix, normalizeIdentifier(fieldName))
}

func normalizeServiceName(name string) string {
	// 规范化服务名，用于文件名
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return reg.ReplaceAllString(name, "_")
}

func normalizeIdentifier(name string) string {
	// 规范化标识符，用于条件名称
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return reg.ReplaceAllString(strings.ToLower(name), "_")
}
