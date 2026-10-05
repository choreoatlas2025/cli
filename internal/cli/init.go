// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	createdFiles, traceRelPath, err := initializeProject(initOptions{
		TargetDir: targetDir, Mode: mode, TracePath: tracePath, Title: title,
		Examples: includeExamples, CI: ciChoice, Force: *forceFlag,
	}, templates.InitFS, os.Rename)
	if err != nil {
		exitErr(err)
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
