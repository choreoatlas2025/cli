// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// ValidateYAMLWithSchema validates YAML file with JSON Schema
func ValidateYAMLWithSchema(yamlPath, schemaPath string) error {
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
	f, err := os.Open(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to open schema file %s: %w", schemaPath, err)
	}
	defer f.Close()

	schemaID := filepath.Base(schemaPath)
	if err := c.AddResource(schemaID, f); err != nil {
		return fmt.Errorf("failed to load schema resource: %w", err)
	}

	sch, err := c.Compile(schemaID)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	// Validate
	if err := sch.Validate(data); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	return nil
}

// ResolvePath resolves path relative to base file
func ResolvePath(basePath, relativePath string) string {
	if filepath.IsAbs(relativePath) {
		return relativePath
	}
	return filepath.Join(filepath.Dir(basePath), relativePath)
}
