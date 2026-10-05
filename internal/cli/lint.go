// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func runLint(args []string) {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec file path")
	useSchema := fs.Bool("schema", true, "Enable JSON Schema strict validation")
	limits := resourceFlags(fs)
	_ = fs.Parse(args)
	if err := checkResourceFlags(*limits); err != nil {
		exitErr(err)
	}

	contract, issues, err := loadContractWithFiles(*flowPath, *useSchema, input.NewSnapshotWithLimit(nil, limits.MaxInputBytes))
	if err != nil {
		exitErr(err)
	}
	config := spec.DefaultValidationConfig()
	config.Limits = *limits
	if _, err := validate.CompilePlan(contract.Flow, contract.Operations, config); err != nil {
		exitErr(err)
	}
	if *useSchema {
		fmt.Println("[SCHEMA] Contract structure validation passed")
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
