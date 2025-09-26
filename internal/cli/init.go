// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/templates"
)

const (
	defaultTemplateTitle = "E-commerce Order Fulfillment Flow"
)

func runInit(args []string) {
	flagSet := flag.NewFlagSet("init", flag.ExitOnError)
	tracePathFlag := flagSet.String("trace", "", "Existing trace.json file path for from-trace mode")
	modeFlag := flagSet.String("mode", "", "Bootstrap mode: template|trace")
	ciFlag := flagSet.String("ci", "", "GitHub Actions workflow template: none|minimal|combo")
	examplesFlag := flagSet.Bool("examples", false, "Copy examples/* directory for reference")
	yesFlag := flagSet.Bool("yes", false, "Accept defaults without interactive prompts")
	forceFlag := flagSet.Bool("force", false, "Overwrite existing files if present")
	outDirFlag := flagSet.String("out", ".", "Target directory (default: current directory)")
	titleFlag := flagSet.String("title", "", "Override FlowSpec title")
	_ = flagSet.Parse(args)

	var (
		examplesProvided bool
		modeProvided     bool
		ciProvided       bool
		traceProvided    bool
	)
	flagSet.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "examples":
			examplesProvided = true
		case "mode":
			modeProvided = true
		case "ci":
			ciProvided = true
		case "trace":
			traceProvided = true
		}
	})

	targetDir := filepath.Clean(*outDirFlag)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		exitErr(fmt.Errorf("failed to ensure target directory %s: %w", targetDir, err))
	}

	interactive := isTerminal(os.Stdin) && !*yesFlag
	reader := bufio.NewReader(os.Stdin)

	mode := strings.ToLower(strings.TrimSpace(*modeFlag))
	if mode == "" {
		if *tracePathFlag != "" {
			mode = "trace"
		} else if interactive && !modeProvided {
			fmt.Println("选择初始化模式:")
			fmt.Println("  1) 模板: 内置电商示例 (默认)")
			fmt.Println("  2) 从现有 trace 生成 FlowSpec / ServiceSpec")
			fmt.Print("请输入选项 [1]: ")
			choice := strings.TrimSpace(readLine(reader))
			if choice == "2" {
				mode = "trace"
			} else {
				mode = "template"
			}
		} else {
			mode = "template"
		}
	}

	if mode != "template" && mode != "trace" {
		exitErr(fmt.Errorf("unsupported init mode: %s", mode))
	}

	tracePath := strings.TrimSpace(*tracePathFlag)
	if mode == "trace" {
		if tracePath == "" && interactive && !traceProvided {
			fmt.Print("请输入 trace.json 路径: ")
			tracePath = strings.TrimSpace(readLine(reader))
		}
		if tracePath == "" {
			exitErr(errors.New("from-trace 模式需要提供 --trace"))
		}
		if _, err := os.Stat(tracePath); err != nil {
			exitErr(fmt.Errorf("无法读取 trace 文件: %w", err))
		}
	}

	flowsDir := filepath.Join(targetDir, "flows")
	servicesDir := filepath.Join(targetDir, "services")
	tracesDir := filepath.Join(targetDir, "traces")
	for _, dir := range []string{flowsDir, servicesDir, tracesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			exitErr(fmt.Errorf("failed to ensure directory %s: %w", dir, err))
		}
	}

	includeExamples := *examplesFlag
	if !examplesProvided && interactive {
		includeExamples = askYesNo(reader, "是否复制完整 examples/* 目录? [y/N]: ", false)
	}

	ciChoice := normalizeCIChoice(strings.ToLower(strings.TrimSpace(*ciFlag)))
	if ciChoice == "" {
		if interactive && !ciProvided {
			fmt.Println("GitHub Actions 模板:")
			fmt.Println("  1) 不生成")
			fmt.Println("  2) 最小版 (lint + validate)")
			fmt.Println("  3) 组合版 (discover + lint + validate + ci-gate)")
			fmt.Print("请选择 [1]: ")
			choice := strings.TrimSpace(readLine(reader))
			switch choice {
			case "2":
				ciChoice = "minimal"
			case "3":
				ciChoice = "combo"
			default:
				ciChoice = "none"
			}
		} else {
			ciChoice = "none"
		}
	}
	if ciChoice == "invalid" {
		exitErr(fmt.Errorf("unknown --ci option: %s", *ciFlag))
	}

	title := strings.TrimSpace(*titleFlag)
	if title == "" {
		if mode == "template" {
			title = defaultTemplateTitle
		} else {
			title = defaultTitleFromTrace(tracePath)
		}
	}

	force := *forceFlag

	var (
		createdFiles []string
		traceRelPath string
		err          error
	)
	switch mode {
	case "template":
		createdFiles, traceRelPath, err = bootstrapFromTemplate(targetDir, flowsDir, servicesDir, tracesDir, title, includeExamples, force)
	case "trace":
		createdFiles, traceRelPath, err = bootstrapFromTrace(targetDir, flowsDir, servicesDir, tracesDir, tracePath, title, includeExamples, force)
	}
	if err != nil {
		exitErr(err)
	}

	if ciChoice != "none" {
		ciFiles, err := injectCIWorkflow(targetDir, ciChoice, force)
		if err != nil {
			exitErr(err)
		}
		createdFiles = append(createdFiles, ciFiles...)
	}

	sort.Strings(createdFiles)

	fmt.Println()
	fmt.Println("✅ ChoreoAtlas init 完成！")
	if len(createdFiles) > 0 {
		fmt.Println("生成/更新的文件:")
		for _, f := range createdFiles {
			fmt.Printf("  - %s\n", f)
		}
	}
	fmt.Println()
	fmt.Println("下一步建议:")
	fmt.Println("  choreoatlas lint")
	if traceRelPath != "" {
		fmt.Printf("  choreoatlas validate --trace %s\n", traceRelPath)
	} else {
		fmt.Println("  choreoatlas validate --trace traces/<your-trace>.json")
	}
	if ciChoice != "none" {
		fmt.Println("  推送到 GitHub 后，choreoatlas.yml 将自动执行")
	}
}

func bootstrapFromTemplate(targetDir, flowsDir, servicesDir, tracesDir, title string, includeExamples, force bool) ([]string, string, error) {
	var created []string

	// Root FlowSpec
	data, err := iofs.ReadFile(templates.InitFS, templates.RootFlowSpecTemplate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read embedded FlowSpec template: %w", err)
	}
	content := bytes.ReplaceAll(data, []byte("__TITLE__"), []byte(title))
	rootFlowPath := filepath.Join(targetDir, ".flowspec.yaml")
	if err := writeTargetFile(rootFlowPath, content, force); err != nil {
		return nil, "", err
	}
	created = append(created, mustRelative(targetDir, rootFlowPath))

	// Flow copy under flows/
	flowDirData, err := iofs.ReadFile(templates.InitFS, templates.FlowDirectorySpecTemplate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read embedded flows template: %w", err)
	}
	flowDirPath := filepath.Join(flowsDir, "order-fulfillment.flowspec.yaml")
	if err := writeTargetFile(flowDirPath, flowDirData, force); err != nil {
		return nil, "", err
	}
	created = append(created, mustRelative(targetDir, flowDirPath))

	// Service specs
	serviceTemplates := []struct {
		src  string
		name string
	}{
		{templates.OrderServiceTemplate, "order-service.servicespec.yaml"},
		{templates.InventoryServiceTemplate, "inventory-service.servicespec.yaml"},
		{templates.ShippingServiceTemplate, "shipping-service.servicespec.yaml"},
	}
	for _, tpl := range serviceTemplates {
		data, err := iofs.ReadFile(templates.InitFS, tpl.src)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read embedded service template %s: %w", tpl.src, err)
		}
		dst := filepath.Join(servicesDir, tpl.name)
		if err := writeTargetFile(dst, data, force); err != nil {
			return nil, "", err
		}
		created = append(created, mustRelative(targetDir, dst))
	}

	// Trace sample
	traceData, err := iofs.ReadFile(templates.InitFS, templates.SuccessfulTraceTemplate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read embedded trace template: %w", err)
	}
	tracePath := filepath.Join(tracesDir, "successful-order.trace.json")
	if err := writeTargetFile(tracePath, traceData, force); err != nil {
		return nil, "", err
	}
	created = append(created, mustRelative(targetDir, tracePath))

	// Optional examples tree
	if includeExamples {
		exampleRoot := filepath.Join(targetDir, "examples")
		copied, err := copyEmbeddedTree(templates.InitFS, templates.ExamplesDir, exampleRoot, force)
		if err != nil {
			return nil, "", err
		}
		created = append(created, copied...)
	}

	return created, mustRelative(targetDir, tracePath), nil
}

func bootstrapFromTrace(targetDir, flowsDir, servicesDir, tracesDir, tracePath, title string, includeExamples, force bool) ([]string, string, error) {
	var created []string

	tr, err := trace.LoadFromFile(tracePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load trace: %w", err)
	}

	sort.Slice(tr.Spans, func(i, j int) bool {
		return tr.Spans[i].StartNanos < tr.Spans[j].StartNanos
	})

	flowContent := generateFlowYAML(tr, title, "./services")
	rootFlowPath := filepath.Join(targetDir, ".flowspec.yaml")
	if err := writeTargetFile(rootFlowPath, []byte(flowContent), force); err != nil {
		return nil, "", err
	}
	created = append(created, mustRelative(targetDir, rootFlowPath))

	flowFileName := deriveFlowFileName(tracePath)
	// 调整 services 相对路径以便 flows/ 内文件可独立 lint
	flowInDir := strings.ReplaceAll(flowContent, "./services/", "../services/")
	flowDirPath := filepath.Join(flowsDir, flowFileName)
	if err := writeTargetFile(flowDirPath, []byte(flowInDir), force); err != nil {
		return nil, "", err
	}
	created = append(created, mustRelative(targetDir, flowDirPath))

	serviceFiles := expectedServiceSpecFiles(servicesDir, tr)
	if err := ensureWritable(serviceFiles, force); err != nil {
		return nil, "", err
	}
	if err := spec.GenerateServiceSpecs(tr.Spans, servicesDir); err != nil {
		return nil, "", fmt.Errorf("failed to generate ServiceSpec files: %w", err)
	}
	for _, path := range serviceFiles {
		created = append(created, mustRelative(targetDir, path))
	}

	// Copy provided trace into traces/
	traceDest := filepath.Join(tracesDir, filepath.Base(tracePath))
	if err := copyFile(tracePath, traceDest, force); err != nil {
		return nil, "", err
	}
	created = append(created, mustRelative(targetDir, traceDest))

	if includeExamples {
		exampleRoot := filepath.Join(targetDir, "examples")
		copied, err := copyEmbeddedTree(templates.InitFS, templates.ExamplesDir, exampleRoot, force)
		if err != nil {
			return nil, "", err
		}
		created = append(created, copied...)
	}

	return created, mustRelative(targetDir, traceDest), nil
}

func injectCIWorkflow(targetDir, choice string, force bool) ([]string, error) {
	var templatePath string
	switch choice {
	case "minimal":
		templatePath = templates.GithubWorkflowMinimalTemplate
	case "combo":
		templatePath = templates.GithubWorkflowComboTemplate
	default:
		return nil, nil
	}

	workflowDir := filepath.Join(targetDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workflows directory: %w", err)
	}
	data, err := iofs.ReadFile(templates.InitFS, templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded workflow template: %w", err)
	}

	dst := filepath.Join(workflowDir, "choreoatlas.yml")
	if err := writeTargetFile(dst, data, force); err != nil {
		return nil, err
	}

	return []string{mustRelative(targetDir, dst)}, nil
}

func copyEmbeddedTree(fsys iofs.FS, srcRoot, dstRoot string, force bool) ([]string, error) {
	var created []string
	err := iofs.WalkDir(fsys, srcRoot, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dstRoot, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := iofs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}
		if err := writeTargetFile(target, data, force); err != nil {
			return err
		}
		created = append(created, target)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i, path := range created {
		created[i] = mustRelative(filepath.Dir(dstRoot), path)
	}
	sort.Strings(created)
	return created, nil
}

func writeTargetFile(path string, data []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("文件已存在: %s (使用 --force 覆盖)", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("无法访问文件 %s: %w", path, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func copyFile(src, dst string, force bool) error {
	if !force {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("文件已存在: %s (使用 --force 覆盖)", dst)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("无法访问文件 %s: %w", dst, err)
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", dst, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dst, err)
	}
	return nil
}

func ensureWritable(paths []string, force bool) error {
	if force {
		return nil
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("文件已存在: %s (使用 --force 覆盖)", p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("无法访问文件 %s: %w", p, err)
		}
	}
	return nil
}

func expectedServiceSpecFiles(servicesDir string, tr *trace.Trace) []string {
	uniq := map[string]struct{}{}
	for _, span := range tr.Spans {
		if span.Service == "" {
			continue
		}
		uniq[span.Service] = struct{}{}
	}
	var paths []string
	for service := range uniq {
		sanitized := normalizeServiceFilename(service)
		filename := fmt.Sprintf("%s.servicespec.yaml", sanitized)
		paths = append(paths, filepath.Join(servicesDir, filename))
	}
	sort.Strings(paths)
	return paths
}

var serviceNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func normalizeServiceFilename(name string) string {
	cleaned := serviceNameSanitizer.ReplaceAllString(name, "_")
	if cleaned == "" {
		return "service"
	}
	return cleaned
}

func deriveFlowFileName(tracePath string) string {
	base := filepath.Base(tracePath)
	trimmed := strings.TrimSuffix(base, filepath.Ext(base))
	if trimmed == "" {
		return "discovered.flowspec.yaml"
	}
	return fmt.Sprintf("%s.flowspec.yaml", trimmed)
}

func defaultTitleFromTrace(tracePath string) string {
	base := filepath.Base(tracePath)
	trimmed := strings.TrimSuffix(base, filepath.Ext(base))
	if trimmed == "" {
		return "Flow discovered from trace"
	}
	return fmt.Sprintf("Flow discovered from %s", trimmed)
}

func normalizeCIChoice(choice string) string {
	switch choice {
	case "", "none":
		return ""
	case "minimal", "combo":
		return choice
	default:
		if choice == "0" {
			return "none"
		}
		return "invalid"
	}
}

func mustRelative(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	if rel == "." {
		return filepath.Base(target)
	}
	return filepath.ToSlash(rel)
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func readLine(reader *bufio.Reader) string {
	line, err := reader.ReadString('\n')
	if err != nil {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(line)
}

func askYesNo(reader *bufio.Reader, prompt string, defaultYes bool) bool {
	fmt.Print(prompt)
	input := strings.ToLower(strings.TrimSpace(readLine(reader)))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}
