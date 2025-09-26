// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package cli

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/choreoatlas2025/cli/internal/schemas"
    "github.com/choreoatlas2025/cli/internal/spec"
    "github.com/choreoatlas2025/cli/internal/validate"
)

// validateAndPersistFlow validates generated FlowSpec YAML (and referenced ServiceSpecs)
// using embedded JSON Schemas and static lint. On success, writes the YAML to outPath.
func validateAndPersistFlow(flowYAML string, outPath string, outServices string) error {
    // Write FlowSpec to a temp file in the same directory as outPath for relative path resolution
    outDir := filepath.Dir(outPath)
    if err := os.MkdirAll(outDir, 0o755); err != nil {
        return fmt.Errorf("failed to ensure output directory: %w", err)
    }
    tmp, err := os.CreateTemp(outDir, ".flowspec.*.yaml")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    tmpPath := tmp.Name()
    if _, err := tmp.WriteString(flowYAML); err != nil {
        tmp.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("failed to write temp flowspec: %w", err)
    }
    tmp.Close()

    // Schema validate FlowSpec
    if err := spec.ValidateYAMLWithSchemaFS(tmpPath, schemas.FS, "flowspec.schema.json"); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("FlowSpec structure validation failed: %w", err)
    }

    // Load flowspec to get service bindings and build op index
    flow, err := spec.LoadFlowSpec(tmpPath)
    if err != nil {
        os.Remove(tmpPath)
        return err
    }

    // Validate each referenced ServiceSpec file using embedded schema
    for alias, bind := range flow.Services {
        serviceSpecPath := spec.ResolvePath(tmpPath, bind.Spec)
        if err := spec.ValidateYAMLWithSchemaFS(serviceSpecPath, schemas.FS, "servicespec.schema.json"); err != nil {
            os.Remove(tmpPath)
            return fmt.Errorf("ServiceSpec structure validation failed (%s): %w", alias, err)
        }
    }

    // Build operation index (loads ServiceSpec logical content)
    _, opIndex, err := flow.BuildOperationIndex(tmpPath)
    if err != nil {
        os.Remove(tmpPath)
        return err
    }

    // Static lint gate (call format, references, variables)
    issues, err := validate.LintFlow(tmpPath, flow, opIndex)
    if err != nil {
        os.Remove(tmpPath)
        return err
    }
    for _, is := range issues {
        if is.Level == "ERROR" {
            os.Remove(tmpPath)
            return fmt.Errorf("lint error: %s", is.Msg)
        }
    }

    // All validations passed; persist to outPath
    if err := os.WriteFile(outPath, []byte(flowYAML), 0o644); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("failed to write flowspec: %w", err)
    }
    os.Remove(tmpPath)
    return nil
}

