// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
)

func runLint(args []string) {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec file path")
	useSchema := fs.Bool("schema", true, "Enable JSON Schema strict validation")
	_ = fs.Parse(args)

	_, issues, err := loadContract(*flowPath, *useSchema)
	if err != nil {
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
