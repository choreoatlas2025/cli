package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
	"github.com/choreoatlas2025/cli/internal/schemas"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func runLint(args []string) {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec file path")
	useSchema := fs.Bool("schema", true, "Enable JSON Schema strict validation")
	_ = fs.Parse(args)

	// JSON Schema validation (if enabled)
	if *useSchema {
		// FlowSpec schema validation (using embedded schema for robustness)
		if err := spec.ValidateYAMLWithSchemaFS(*flowPath, schemas.FS, "flowspec.schema.json"); err != nil {
			// Fallback to file path method
			if err := spec.ValidateYAMLWithSchema(*flowPath, "schemas/flowspec.schema.json"); err != nil {
				exitErr(fmt.Errorf("FlowSpec structure validation failed: %w", err))
			}
		}
		fmt.Println("[SCHEMA] FlowSpec structure validation passed")
	}

	flow, err := spec.LoadFlowSpec(*flowPath)
	if err != nil {
		exitErr(err)
	}

	// ServiceSpec schema validation (if enabled)
	if *useSchema {
		for alias, bind := range flow.Services {
			serviceSpecPath := spec.ResolvePath(*flowPath, bind.Spec)
			// Using embedded schema for robustness
			if err := spec.ValidateYAMLWithSchemaFS(serviceSpecPath, schemas.FS, "servicespec.schema.json"); err != nil {
				// Fallback to file path method
				if err := spec.ValidateYAMLWithSchema(serviceSpecPath, "schemas/servicespec.schema.json"); err != nil {
					exitErr(fmt.Errorf("ServiceSpec structure validation failed (%s): %w", alias, err))
				}
			}
		}
		fmt.Println("[SCHEMA] ServiceSpec structure validation passed")
	}
	_, opIndex, err := flow.BuildOperationIndex(*flowPath)
	if err != nil {
		exitErr(err)
	}
	issues, err := validate.LintFlow(*flowPath, flow, opIndex)
	if err != nil {
		exitErr(err)
	}

	if len(issues) == 0 {
		fmt.Println("Lint: OK")
		return
	}
	errCount := 0
	for _, is := range issues {
		if is.Level == "ERROR" {
			errCount++
		}
		fmt.Printf("[%s] %s\n", is.Level, is.Msg)
	}
	if errCount > 0 {
		os.Exit(exitcode.InputError)
	}
}
