package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// ValidateYAMLWithSchema 使用 JSON Schema 验证 YAML 文件
func ValidateYAMLWithSchema(yamlPath, schemaPath string) error {
	// 读 YAML -> JSON 兼容的 map
	b, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("读取文件 %s: %w", yamlPath, err)
	}
	var data any
	if err := yaml.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}

	// 编译 schema
	c := jsonschema.NewCompiler()
	f, err := os.Open(schemaPath)
	if err != nil {
		return fmt.Errorf("打开 schema 文件 %s: %w", schemaPath, err)
	}
	defer f.Close()

	schemaID := filepath.Base(schemaPath)
	if err := c.AddResource(schemaID, f); err != nil {
		return fmt.Errorf("加载 schema 资源: %w", err)
	}

	sch, err := c.Compile(schemaID)
	if err != nil {
		return fmt.Errorf("编译 schema: %w", err)
	}

	// 校验
	if err := sch.Validate(data); err != nil {
		return fmt.Errorf("schema 校验失败: %w", err)
	}

	return nil
}

// ResolvePath 解析相对于基础文件的路径
func ResolvePath(basePath, relativePath string) string {
	if filepath.IsAbs(relativePath) {
		return relativePath
	}
	return filepath.Join(filepath.Dir(basePath), relativePath)
}
