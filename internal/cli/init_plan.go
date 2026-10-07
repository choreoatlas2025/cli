// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choreoatlas2025/cli/internal/discovery"
	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/schemas"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
	"github.com/choreoatlas2025/cli/templates"
	"gopkg.in/yaml.v3"
)

type initOptions struct {
	TargetDir, Mode, TracePath, Title, CI string
	Examples, Force                       bool
}

// Render and qualify the complete generation set before any destination is
// replaced. The commit helper rolls back ordinary errors, including CI files.
func initializeProject(opts initOptions, assets fs.FS, rename func(string, string) error) ([]string, string, error) {
	files, tracePath, err := renderInitFiles(opts, assets)
	if err != nil {
		return nil, "", err
	}
	if err := qualifyInitFiles(files, tracePath); err != nil {
		return nil, "", fmt.Errorf("invalid init output: %w", err)
	}
	if !opts.Force {
		for _, file := range files {
			if _, err := os.Lstat(file.path); err == nil {
				return nil, "", fmt.Errorf("文件已存在: %s (使用 --force 覆盖)", file.path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, "", err
			}
		}
	}
	if err := commitGeneratedFiles(files, rename); err != nil {
		return nil, "", err
	}
	created := make([]string, len(files))
	for i, file := range files {
		created[i] = mustRelative(opts.TargetDir, file.path)
	}
	return created, mustRelative(opts.TargetDir, tracePath), nil
}

func renderInitFiles(opts initOptions, assets fs.FS) ([]generatedFile, string, error) {
	var files []generatedFile
	add := func(relative string, data []byte) {
		files = append(files, generatedFile{path: filepath.Join(opts.TargetDir, filepath.FromSlash(relative)), data: data})
	}
	copyAsset := func(src, relative string) error {
		data, err := fs.ReadFile(assets, src)
		if err == nil {
			add(relative, data)
		}
		return err
	}
	var traceRelative string
	switch opts.Mode {
	case "template":
		for _, file := range []struct{ src, dest string }{
			{templates.RootFlowSpecTemplate, ".flowspec.yaml"},
			{templates.FlowDirectorySpecTemplate, "flows/order-fulfillment.flowspec.yaml"},
		} {
			data, err := fs.ReadFile(assets, file.src)
			if err != nil {
				return nil, "", err
			}
			flow, err := spec.ParseFlowSpec(data)
			if err != nil {
				return nil, "", err
			}
			flow.Info.Title = opts.Title
			data, err = yaml.Marshal(flow)
			if err != nil {
				return nil, "", err
			}
			add(file.dest, data)
		}
		for _, file := range []struct{ src, dest string }{
			{templates.OrderServiceTemplate, "services/order-service.servicespec.yaml"},
			{templates.InventoryServiceTemplate, "services/inventory-service.servicespec.yaml"},
			{templates.ShippingServiceTemplate, "services/shipping-service.servicespec.yaml"},
			{templates.SuccessfulTraceTemplate, "traces/successful-order.trace.json"},
		} {
			if err := copyAsset(file.src, file.dest); err != nil {
				return nil, "", err
			}
		}
		traceRelative = "traces/successful-order.trace.json"
	case "trace":
		captured, err := input.NewSnapshot(nil).Read(opts.TracePath)
		if err != nil {
			return nil, "", err
		}
		tr, err := trace.Parse(captured.Bytes())
		if err != nil {
			return nil, "", err
		}
		if err := trace.ValidateTimestamps(tr.Spans, true); err != nil {
			return nil, "", err
		}
		for _, output := range []struct{ path, services string }{
			{".flowspec.yaml", "./services"},
			{"flows/" + deriveFlowFileName(opts.TracePath), "../services"},
		} {
			generated, err := discovery.FlowYAML(tr, opts.Title, output.services)
			if err != nil {
				return nil, "", err
			}
			add(output.path, []byte(generated))
		}
		services, err := discovery.BuildServiceSpecFiles(tr.Spans)
		if err != nil {
			return nil, "", err
		}
		names := make([]string, 0, len(services))
		for name := range services {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			add("services/"+name, services[name])
		}
		traceRelative = "traces/" + filepath.Base(opts.TracePath)
		// Parsing, generation and the copied input all consume this capture.
		add(traceRelative, captured.Bytes())
	default:
		return nil, "", fmt.Errorf("unsupported init mode: %s", opts.Mode)
	}
	if opts.Examples {
		if err := fs.WalkDir(assets, templates.ExamplesDir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			return copyAsset(path, "examples/"+strings.TrimPrefix(path, templates.ExamplesDir+"/"))
		}); err != nil {
			return nil, "", err
		}
	}
	switch opts.CI {
	case "none", "":
	case "minimal", "combo":
		template := templates.GithubWorkflowMinimalTemplate
		if opts.CI == "combo" {
			template = templates.GithubWorkflowComboTemplate
		}
		if err := copyAsset(template, ".github/workflows/choreoatlas.yml"); err != nil {
			return nil, "", err
		}
	default:
		return nil, "", fmt.Errorf("unsupported CI template: %s", opts.CI)
	}
	return files, filepath.Join(opts.TargetDir, filepath.FromSlash(traceRelative)), nil
}

func qualifyInitFiles(files []generatedFile, primaryTrace string) error {
	generated := make(map[string][]byte, len(files))
	for _, file := range files {
		path, err := filepath.Abs(file.path)
		if err != nil {
			return err
		}
		if _, exists := generated[path]; exists {
			return fmt.Errorf("duplicate generation destination: %s", path)
		}
		generated[path] = file.data
	}
	// Existing files cannot mask missing or invalid generated references.
	capture := input.NewSnapshot(func(path string) ([]byte, error) {
		data, exists := generated[path]
		if !exists {
			return nil, fmt.Errorf("reference outside generation set: %s", path)
		}
		return data, nil
	})
	for _, file := range files {
		var err error
		switch {
		case file.path == primaryTrace:
			var tr *trace.Trace
			tr, err = trace.Parse(file.data)
			if err == nil {
				err = trace.ValidateTimestamps(tr.Spans, true)
			}
		case strings.HasSuffix(file.path, ".flowspec.yaml"):
			contract, loadErr := loadAndValidateContract(file.path, capture)
			err = loadErr
			if err == nil {
				_, err = validate.CompilePlan(contract.Flow, contract.Operations, spec.DefaultValidationConfig())
			}
		case strings.HasSuffix(file.path, ".servicespec.yaml"):
			err = spec.ValidateYAMLBytesWithSchemaFS(file.data, schemas.FS, "servicespec.schema.json")
			if err == nil {
				_, err = spec.ParseServiceSpec(file.data)
			}
		case strings.HasSuffix(file.path, ".trace.json"):
			_, err = trace.Parse(file.data)
		case strings.HasSuffix(file.path, ".yml"):
			var workflow any
			err = yaml.Unmarshal(file.data, &workflow)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", file.path, err)
		}
	}
	return nil
}
