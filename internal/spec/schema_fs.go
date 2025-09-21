// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// ValidateYAMLWithSchemaFS validates YAML file with embedded JSON Schema (recommended)
func ValidateYAMLWithSchemaFS(yamlPath string, fsys fs.FS, schemaName string) error {
	// Read YAML -> JSON compatible map
	b, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", yamlPath, err)
	}
	var data any
	if err := yaml.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Compile schema
	c := jsonschema.NewCompiler()
	f, err := fsys.Open(schemaName)
	if err != nil {
		return fmt.Errorf("failed to open embedded schema %s: %w", schemaName, err)
	}
	defer f.Close()
	
	if err := c.AddResource(schemaName, f); err != nil {
		return fmt.Errorf("failed to load embedded schema resource: %w", err)
	}
	
	sch, err := c.Compile(schemaName)
	if err != nil {
		return fmt.Errorf("failed to compile embedded schema: %w", err)
	}
	
	// Validate
	if err := sch.Validate(data); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	
	return nil
}