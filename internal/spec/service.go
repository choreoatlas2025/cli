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
	return ParseServiceSpec(b)
}

func ParseServiceSpec(b []byte) (*ServiceSpecFile, error) {
	var ss ServiceSpecFile
	if err := yaml.Unmarshal(b, &ss); err != nil {
		return nil, fmt.Errorf("failed to parse servicespec: %w", err)
	}
	seen := map[string]bool{}
	for _, op := range ss.Operations {
		if seen[op.OperationId] {
			return nil, fmt.Errorf("invalid ServiceSpec: duplicate operationId %q", op.OperationId)
		}
		seen[op.OperationId] = true
	}
	return &ss, nil
}

// BuildServiceSpecFiles generates data without changing any destination files.
func BuildServiceSpecFiles(spans []trace.Span) (map[string][]byte, error) {
	groups, err := groupSpansByService(spans)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for service, operations := range groups {
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
func groupSpansByService(spans []trace.Span) (map[string][]ServiceOperation, error) {
	serviceOps := make(map[string][]ServiceOperation)

	// 按服务和操作分组
	opGroups := make(map[string]map[string][]trace.Span)
	sources := make(map[string]map[string]string)

	for _, span := range spans {
		if span.Service == "" || span.Name == "" {
			continue
		}

		service := span.Service
		opName := ComputeOperationID(span)

		if _, exists := opGroups[service]; !exists {
			opGroups[service] = make(map[string][]trace.Span)
			sources[service] = make(map[string]string)
		}
		source := operationSource(span)
		if previous, exists := sources[service][opName]; exists && previous != source {
			return nil, fmt.Errorf("invalid operation identity: service %q operationId %q merges %q and %q", service, opName, previous, source)
		}
		sources[service][opName] = source
		opGroups[service][opName] = append(opGroups[service][opName], span)
	}

	// 为每个服务的每个操作生成 ServiceOperation
	for service, ops := range opGroups {
		var operations []ServiceOperation
		for opName, spanList := range ops {
			op := generateServiceOperation(opName, spanList)
			operations = append(operations, op)
		}
		serviceOps[service] = operations
	}

	return serviceOps, nil
}

// generateServiceOperation retains only observations shared by all instances.
// Repeated calls and retries may have different responses: the last observed
// value cannot become a requirement imposed on every instance.
func generateServiceOperation(opName string, spans []trace.Span) ServiceOperation {
	preconditions, postconditions := map[string]string{}, map[string]string{}
	varied := false
	for i, span := range spans {
		pre, post := map[string]string{}, map[string]string{}
		for _, key := range sortedAttributeKeys(span.Attributes) {
			value := span.Attributes[key]
			if expr := buildCELExpression(key, value); expr != "" {
				if isRequestAttribute(key) {
					pre[generateConditionName("req", key)] = expr
				} else if isResponseAttribute(key) {
					post[generateConditionName("resp", key)] = expr
				}
			}
		}
		if i == 0 {
			preconditions, postconditions = pre, post
			continue
		}
		for _, pair := range [][2]map[string]string{{preconditions, pre}, {postconditions, post}} {
			for key, expr := range pair[0] {
				if pair[1][key] != expr {
					delete(pair[0], key)
					varied = true
				}
			}
			for key, expr := range pair[1] {
				if pair[0][key] != expr {
					varied = true
				}
			}
		}
	}
	description := fmt.Sprintf("Auto-generated %s operation from trace", opName)
	if varied {
		description += "; varying or missing observations omitted: review retry outcomes and add business requirements"
	}
	return ServiceOperation{OperationId: opName, Description: description, Preconditions: preconditions, Postconditions: postconditions}
}

func sortedAttributeKeys(attributes map[string]any) []string {
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// buildCELExpression 根据属性键值生成 CEL 表达式
func buildCELExpression(key string, value interface{}) string {
	keyLower := strings.ToLower(key)

	// Special cases for common HTTP request attributes
	if keyLower == "http.method" || keyLower == "http.request.method" {
		if s, ok := value.(string); ok && s != "" {
			return fmt.Sprintf("span.attributes[%q] == %q", key, s)
		}
	}
	if keyLower == "http.route" || keyLower == "http.target" || keyLower == "http.url" || keyLower == "url.path" {
		if s, ok := value.(string); ok && s != "" {
			return fmt.Sprintf("span.attributes[%q] == %q", key, s)
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
		return fmt.Sprintf("span.attributes[%q].matches(\"Bearer .+\")", key)
	}

	// 字符串非空检查
	if strVal, ok := value.(string); ok && strVal != "" {
		if strings.Contains(keyLower, "request") || strings.Contains(keyLower, "body") {
			return fmt.Sprintf("span.attributes[%q] != \"\"", key)
		} else if strings.Contains(keyLower, "response") {
			return fmt.Sprintf("span.attributes[%q] != \"\"", key)
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
	requestPatterns := []string{"request.", "http.method", "http.url", "http.route", "http.target", "url.path"}

	for _, pattern := range requestPatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}
	return false
}

func isResponseAttribute(key string) bool {
	keyLower := strings.ToLower(key)
	responsePatterns := []string{"response.", "http.status_code", "http.response.status_code"}

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
