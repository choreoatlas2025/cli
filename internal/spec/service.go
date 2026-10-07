// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
	"fmt"
	"os"

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
