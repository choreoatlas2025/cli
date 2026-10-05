// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io/fs"

	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/schemas"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func loadContract(flowPath string, useSchema bool) (*spec.ContractSnapshot, []validate.LintIssue, error) {
	return loadContractWithFiles(flowPath, useSchema, input.NewSnapshot(nil))
}

func loadContractWithFiles(flowPath string, useSchema bool, files *input.Snapshot) (*spec.ContractSnapshot, []validate.LintIssue, error) {
	var schemaFS fs.FS
	if useSchema {
		schemaFS = schemas.FS
	}
	contract, err := spec.LoadContractSnapshot(flowPath, files, schemaFS)
	if err != nil {
		return nil, nil, err
	}
	issues, err := validate.LintFlow(flowPath, contract.Flow, contract.Operations)
	return contract, issues, err
}

func loadTraceSnapshot(path string, files *input.Snapshot) (*trace.Trace, string, error) {
	file, err := files.Read(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read trace file: %w", err)
	}
	tr, err := trace.Parse(file.Bytes())
	if err != nil {
		return nil, "", err
	}
	return tr, file.Hash(), nil
}
