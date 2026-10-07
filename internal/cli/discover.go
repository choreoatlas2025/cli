// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"
	"fmt"

	"github.com/choreoatlas2025/cli/internal/trace"
)

func runDiscover(args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	tracePath := fs.String("trace", "", "trace.json file path")
	out := fs.String("out", "discovered.flowspec.yaml", "FlowSpec output path")
	outServices := fs.String("out-services", "./services", "ServiceSpec output directory")
	title := fs.String("title", "Flow generated from trace", "FlowSpec title")
	noValidate := fs.Bool("no-validate", false, "Skip schema + lint validation gate (not recommended)")
	_ = fs.Parse(args)

	if *tracePath == "" {
		exitErr(fmt.Errorf("--trace parameter is required"))
	}

	tr, err := trace.LoadFromFile(*tracePath)
	if err != nil {
		exitErr(err)
	}

	if err := discoverAndPersist(tr, *title, *out, *outServices, *noValidate); err != nil {
		exitErr(err)
	}
	fmt.Printf("Generated FlowSpec: %s\n", *out)

	fmt.Println("Dual contract generation complete! Please adjust the generated specifications as needed.")
}
