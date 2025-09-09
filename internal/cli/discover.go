package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func runDiscover(args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	tracePath := fs.String("trace", "", "trace.json 文件路径")
	out := fs.String("out", "discovered.flowspec.yaml", "生成的 FlowSpec 输出路径")
	outServices := fs.String("out-services", "./services", "ServiceSpec 文件输出目录")
	title := fs.String("title", "从 Trace 探索生成的流程", "生成的 FlowSpec 标题")
	_ = fs.Parse(args)

	if *tracePath == "" {
		exitErr(fmt.Errorf("--trace 参数是必须的"))
	}

	tr, err := trace.LoadFromFile(*tracePath)
	if err != nil {
		exitErr(err)
	}

	// 按时间排序 spans
	sort.Slice(tr.Spans, func(i, j int) bool {
		return tr.Spans[i].StartNanos < tr.Spans[j].StartNanos
	})

	// 生成 FlowSpec YAML
	yml := generateFlowYAML(tr, *title, *outServices)
	if err := os.WriteFile(*out, []byte(yml), 0644); err != nil {
		exitErr(fmt.Errorf("写入文件失败: %w", err))
	}

	fmt.Printf("已生成 FlowSpec: %s\n", *out)

	// 生成 ServiceSpec 文件
	if err := spec.GenerateServiceSpecs(tr.Spans, *outServices); err != nil {
		exitErr(fmt.Errorf("生成 ServiceSpec 失败: %w", err))
	}

	fmt.Println("双契约生成完成！请根据需要调整生成的规约。")
}

// generateFlowYAML 从 trace 生成 FlowSpec YAML
func generateFlowYAML(tr *trace.Trace, title string, outServices string) string {
	var sb strings.Builder

	// Info 部分
	sb.WriteString("info:\n")
	sb.WriteString(fmt.Sprintf("  title: \"%s\"\n\n", title))

	// Services 部分
	services := make(map[string]struct{})
	for _, span := range tr.Spans {
		if span.Service != "" {
			services[span.Service] = struct{}{}
		}
	}

	sb.WriteString("services:\n")
	for service := range services {
		sb.WriteString(fmt.Sprintf("  %s:\n", service))
		sb.WriteString(fmt.Sprintf("    spec: \"%s/%s.servicespec.yaml\"\n", outServices, service))
	}
	sb.WriteString("\n")

	// Flow 部分
	sb.WriteString("flow:\n")
	for i, span := range tr.Spans {
		stepName := fmt.Sprintf("步骤%d-%s", i+1, span.Name)
		if span.Service == "" || span.Name == "" {
			continue // 跳过无效的 span
		}

		sb.WriteString(fmt.Sprintf("  - step: \"%s\"\n", stepName))
		sb.WriteString(fmt.Sprintf("    call: \"%s.%s\"\n", span.Service, span.Name))

		// 生成示例 input（基于 attributes）
		if len(span.Attributes) > 0 {
			sb.WriteString("    input:\n")
			sb.WriteString("      body:\n")
			for key, value := range span.Attributes {
				// 简单的变量引用推断
				if strings.Contains(key, "Id") {
					sb.WriteString(fmt.Sprintf("        %s: \"${%s}\"  # TODO: 检查变量引用\n", key, key))
				} else {
					sb.WriteString(fmt.Sprintf("        %s: %v  # TODO: 调整输入值\n", key, value))
				}
			}
		}

		// 生成 output（假设每个步骤都有响应）
		outputVar := fmt.Sprintf("%sResponse", strings.ToLower(span.Service))
		sb.WriteString("    output:\n")
		sb.WriteString(fmt.Sprintf("      %s: \"response.body\"  # TODO: 调整输出映射\n", outputVar))

		sb.WriteString("\n")
	}

	// 添加注释说明
	sb.WriteString("# 此文件由 flowspec discover 自动生成\n")
	sb.WriteString("# TODO 列表：\n")
	sb.WriteString("# 1. 创建对应的 ServiceSpec 文件\n")
	sb.WriteString("# 2. 调整输入输出映射和变量引用\n")
	sb.WriteString("# 3. 验证步骤顺序和调用关系\n")
	sb.WriteString("# 4. 添加适当的 meta 信息\n")

	return sb.String()
}
