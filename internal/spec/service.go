package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/choreoatlas2025/cli/internal/trace"
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
		return nil, fmt.Errorf("读取 servicespec 失败: %w", err)
	}
	var ss ServiceSpecFile
	if err := yaml.Unmarshal(b, &ss); err != nil {
		return nil, fmt.Errorf("解析 servicespec 失败: %w", err)
	}
	return &ss, nil
}

// GenerateServiceSpecs 从 trace spans 生成 ServiceSpec 文件
func GenerateServiceSpecs(spans []trace.Span, outDir string) error {
	// 按服务分组操作
	serviceOps := groupSpansByService(spans)
	
	// 确保输出目录存在
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	
	// 为每个服务生成 ServiceSpec 文件
	for serviceName, operations := range serviceOps {
		spec := &ServiceSpecFile{
			Service:    serviceName,
			Operations: operations,
		}
		
		// 序列化为 YAML
		data, err := yaml.Marshal(spec)
		if err != nil {
			return fmt.Errorf("序列化服务 %s 的 ServiceSpec 失败: %w", serviceName, err)
		}
		
		// 写入文件
		filename := fmt.Sprintf("%s.servicespec.yaml", normalizeServiceName(serviceName))
		filePath := filepath.Join(outDir, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("写入 ServiceSpec 文件 %s 失败: %w", filePath, err)
		}
		
		fmt.Printf("已生成 ServiceSpec: %s\n", filePath)
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
		opName := normalizeOperationName(span.Name)
		
		if _, exists := opGroups[service]; !exists {
			opGroups[service] = make(map[string][]trace.Span)
		}
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
		Description:    fmt.Sprintf("从追踪自动生成的 %s 操作", opName),
		Preconditions:  preconditions,
		Postconditions: postconditions,
	}
}

// buildCELExpression 根据属性键值生成 CEL 表达式
func buildCELExpression(key string, value interface{}) string {
	keyLower := strings.ToLower(key)
	
	// 状态码检查
	if isStatusAttribute(key, value) {
		if intVal, ok := value.(int); ok {
			return fmt.Sprintf("response.status == %d", intVal)
		} else if floatVal, ok := value.(float64); ok {
			return fmt.Sprintf("response.status == %d", int(floatVal))
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
	requestPatterns := []string{"request.", "http.method", "http.url"}
	
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

func normalizeOperationName(name string) string {
	// 规范化操作名，转为驼峰式
	// 简单实现：去除特殊字符，首字母小写
	reg := regexp.MustCompile(`[^a-zA-Z0-9]`)
	clean := reg.ReplaceAllString(name, "")
	if len(clean) > 0 {
		return strings.ToLower(clean[:1]) + clean[1:]
	}
	return clean
}

func normalizeIdentifier(name string) string {
	// 规范化标识符，用于条件名称
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return reg.ReplaceAllString(strings.ToLower(name), "_")
}
