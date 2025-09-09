package spec

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// ValidateYAMLWithSchemaFS 使用嵌入式 JSON Schema 验证 YAML 文件（推荐）
func ValidateYAMLWithSchemaFS(yamlPath string, fsys fs.FS, schemaName string) error {
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
	f, err := fsys.Open(schemaName)
	if err != nil {
		return fmt.Errorf("打开嵌入式 schema %s: %w", schemaName, err)
	}
	defer f.Close()
	
	if err := c.AddResource(schemaName, f); err != nil {
		return fmt.Errorf("加载嵌入式 schema 资源: %w", err)
	}
	
	sch, err := c.Compile(schemaName)
	if err != nil {
		return fmt.Errorf("编译嵌入式 schema: %w", err)
	}
	
	// 校验
	if err := sch.Validate(data); err != nil {
		return fmt.Errorf("schema 校验失败: %w", err)
	}
	
	return nil
}