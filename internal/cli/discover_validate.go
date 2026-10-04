// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"gopkg.in/yaml.v3"
)

func validateGeneratedFlow(path string) error {
	_, _, err := loadAndValidateFlow(path)
	return err
}

// validateAndPersistFlow retains the single-file helper used by discovery tests.
func validateAndPersistFlow(flowYAML, outPath, _ string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	staged, err := stageGeneratedFile(outPath, []byte(flowYAML), 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(staged) }()
	if err := validateGeneratedFlow(staged); err != nil {
		return err
	}
	return commitGeneratedFiles([]generatedFile{{path: outPath, data: []byte(flowYAML)}}, os.Rename)
}

func discoverAndPersist(tr *trace.Trace, title, outPath, outServices string, noValidate bool) error {
	flowPath, err := filepath.Abs(outPath)
	if err != nil {
		return err
	}
	servicesDir, err := filepath.Abs(outServices)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(flowPath), 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(filepath.Dir(flowPath), ".choreoatlas-discover-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	files, err := spec.BuildServiceSpecFiles(tr.Spans)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(staging, name), data, 0o644); err != nil {
			return err
		}
		names = append(names, name)
	}
	sort.Strings(names)
	// Both staged and final contracts refer to files relative to their own paths.
	if !noValidate {
		stagedFlow := filepath.Join(staging, "flow.yaml")
		if err := os.WriteFile(stagedFlow, []byte(generateFlowYAML(tr, title, staging)), 0o644); err != nil {
			return err
		}
		if err := validateGeneratedFlow(stagedFlow); err != nil {
			return err
		}
	}
	bindingDir, err := filepath.Rel(filepath.Dir(flowPath), servicesDir)
	if err != nil {
		return err
	}
	flowYAML := generateFlowYAML(tr, title, bindingDir)
	// Final output must remain parseable even when schema/lint checks are skipped.
	var flow spec.FlowSpec
	if err := yaml.Unmarshal([]byte(flowYAML), &flow); err != nil {
		return err
	}
	var updates []generatedFile
	for _, name := range names {
		updates = append(updates, generatedFile{path: filepath.Join(servicesDir, name), data: files[name]})
	}
	updates = append(updates, generatedFile{path: flowPath, data: []byte(flowYAML)})
	if err := commitGeneratedFiles(updates, os.Rename); err != nil {
		return err
	}
	for _, name := range names {
		fmt.Printf("Generated ServiceSpec: %s\n", filepath.Join(outServices, name))
	}
	return nil
}
