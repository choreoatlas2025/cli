#!/bin/bash

# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0
# CE v0.7.0-ce Setup Script with Auto Issue URL Backfill
# 使用前请修改以下变量

# === 必填 ===
export REPO="choreoatlas2025/cli"  # CLI 代码仓库
export MILESTONE="CE v0.7.0-ce"    # 首个 CE 版本里程碑
export DEFAULT_BRANCH="main"       # 默认分支名
# 审阅人（可选，留空则不指定）
export REVIEWERS=""                # 留空，如需要再填

# 存储 Issue ID 的临时文件
ISSUE_MAP_FILE="/tmp/ce_issue_map_$$.txt"
> "$ISSUE_MAP_FILE"

# 辅助函数：创建 Issue 并记录 ID
create_issue_and_record() {
  local key="$1"
  shift
  local issue_url=$(gh issue create "$@")
  local issue_id=$(echo "$issue_url" | grep -o '[0-9]*$')
  echo "$key=$issue_id" >> "$ISSUE_MAP_FILE"
  echo "Created: $key -> #$issue_id ($issue_url)"
}


# 辅助函数：获取记录的 Issue ID
get_issue_id() {
  local key="$1"
  grep "^$key=" "$ISSUE_MAP_FILE" | cut -d= -f2
}

echo "========================================="
echo "CE v0.7.0-ce 设置脚本"
echo "========================================="

# Step 0: 设置默认仓库
echo "设置默认仓库: $REPO"
gh repo set-default "$REPO"

# Step 1: 创建里程碑和标签
echo ""
echo "Step 1: 创建里程碑和标签..."

gh milestone create "$MILESTONE" -d "Community Edition 0.7.0-ce cut (code+docs+release)" 2>/dev/null || echo "里程碑可能已存在"

for L in "P0" "P1" "P2" \
         "area:cli" "area:validate" "area:schema" "area:report" "area:release" "area:docs" \
         "type:feat" "type:fix" "type:docs" "type:refactor" "type:ci" \
         "CE"; do
  gh label create "$L" --force >/dev/null 2>&1
done

echo "标签创建完成"

# Step 2: 创建所有子 Issues
echo ""
echo "Step 2: 创建子 Issues..."

# PR#1 子 Issues
create_issue_and_record "pr1_sub1" \
  --title "internal/cli/exitcode：统一退出码常量（0–4）" \
  --body $'动机：消除魔法数字；与文档对齐。\n验收：引入 internal/cli/exitcode；常量含注释；godoc 可读。' \
  --label "P0,area:cli,type:fix,CE" --milestone "$MILESTONE"

create_issue_and_record "pr1_sub2" \
  --title "validate 命令改用 exitcode 常量并清除分歧（含门控失败=4）" \
  --body $'动机：与 README 对齐，避免 CI 漏判。\n验收：门控失败统一 os.Exit(4)；其余路径不变。' \
  --label "P0,area:cli,type:fix,CE" --milestone "$MILESTONE"

create_issue_and_record "pr1_sub3" \
  --title "CLI 集成测试：覆盖各种失败码（schema/assert/gate）" \
  --body $'动机：防回归；覆盖 0–4。\n验收：新增 e2e/cli 测试，断言退出码与文案。' \
  --label "P0,area:cli,type:ci,CE" --milestone "$MILESTONE"

create_issue_and_record "pr1_sub4" \
  --title "文档：README/CLI 参考更新退出码 0–4 表" \
  --body $'动机：现文档 0–3 不准确。\n验收：README 与 /docs/reference/cli/validate.md 同时更新并互相链接。' \
  --label "P0,area:docs,type:docs,CE" --milestone "$MILESTONE"

# PR#2 子 Issues
create_issue_and_record "pr2_sub1" \
  --title "CLI：--baseline 传递到 EvaluateGate（消除读取后丢弃）" \
  --body $'动机：兑现基线能力。\n验收：第三参数不为 nil；无基线不回归。路径：internal/cli/validate.go' \
  --label "P0,area:cli,type:fix,CE" --milestone "$MILESTONE"

create_issue_and_record "pr2_sub2" \
  --title "EvaluateGate：实现 relative 模式与比较运算" \
  --body $'动机：支持相对阈值门控。\n验收：delta_pct 计算正确；阈值边界用例通过。' \
  --label "P0,area:validate,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr2_sub3" \
  --title "报告：输出 baseline_value / delta_abs / delta_pct（HTML/JSON/JUnit）" \
  --body $'动机：让门控理由可见。\n验收：三种格式均包含字段；golden 快照更新。' \
  --label "P0,area:report,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr2_sub4" \
  --title "CLI：--baseline-missing 策略（fail|treat-as-absolute）" \
  --body $'动机：控制缺失基线时的行为。\n验收：两策略均有效；默认 fail；文档同步。' \
  --label "P0,area:cli,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr2_sub5" \
  --title "文档：基线+阈值 说明与报告截图" \
  --body $'动机：教用户读报告。\n验收：/docs/reports/ce-report.md 更新并含截图。' \
  --label "P0,area:docs,type:docs,CE" --milestone "$MILESTONE"

# PR#3 子 Issues
create_issue_and_record "pr3_sub1" \
  --title "CheckCausality：基于 graph 边的偏序与约束实现" \
  --body $'动机：兑现"编排级时序/因果/DAG 验证"。\n验收：按边检查，返回违规边列表，附时间容差逻辑。' \
  --label "P0,area:validate,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr3_sub2" \
  --title "validateWithCausality：并发/OTLP 路径对齐调用新校验器" \
  --body $'动机：统一校验路径。\n验收：dynamic.go 使用新 API；集成测试覆盖。' \
  --label "P0,area:validate,type:fix,CE" --milestone "$MILESTONE"

create_issue_and_record "pr3_sub3" \
  --title "CLI：--causality-tolerance 开关（默认 10–50ms）" \
  --body $'动机：适配不同系统抖动。\n验收：不同容差给出相应判定差异。' \
  --label "P1,area:cli,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr3_sub4" \
  --title "文档：因果/DAG 校验规则与示例（含违规输出）" \
  --body $'动机：让用户理解结果。\n验收：/docs/flowspec/causality.md 更新，含两张图与输出示例。' \
  --label "P0,area:docs,type:docs,CE" --milestone "$MILESTONE"

# PR#4 子 Issues
create_issue_and_record "pr4_sub1" \
  --title "对外 Schema：增加 oneOf(graph|flow) 与稳定 \$id" \
  --body $'动机：用户能直接引用仓库 schema。\n验收：使用任一入口均通过；\$id 可被编辑器识别缓存。' \
  --label "P0,area:schema,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr4_sub2" \
  --title "解析器：优先 graph，失败回退 flow（一次性警告）" \
  --body $'动机：平滑迁移。\n验收：混合项目可运行；旧 flow 触发一次性 deprecate 提示。' \
  --label "P2,area:schema,type:fix,CE" --milestone "$MILESTONE"

create_issue_and_record "pr4_sub3" \
  --title "文档：Schema 使用（graph 默认、oneOf 兼容、编辑器绑定）" \
  --body $'动机：降低上手难度。\n验收：/docs/flowspec/schema.md 更新，附 JSON 示例。' \
  --label "P0,area:docs,type:docs,CE" --milestone "$MILESTONE"

# PR#5 子 Issues (调整为验收现有功能)
create_issue_and_record "pr5_sub1" \
  --title "验收：build tags 剔除 telemetry/org 代码路径（CE 下 no-op）" \
  --body $'动机：验证隐私与体积优化已实现。\n验收：nm/SBOM 不含相关符号；增补缺失的 stub。' \
  --label "P1,area:release,type:refactor,CE" --milestone "$MILESTONE"

create_issue_and_record "pr5_sub2" \
  --title "验收：--version 输出 vX.Y.Z-ce；报告页角标 CE" \
  --body $'动机：验证对外可见差异已实现。\n验收：测试覆盖；截图/快照更新。' \
  --label "P1,area:report,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr5_sub3" \
  --title "文档：安装（Release/Homebrew/Docker）与零遥测声明" \
  --body $'动机：闭环新用户路径。\n验收：三种安装命令可复制；隐私页明确"零遥测"。' \
  --label "P1,area:docs,type:docs,CE" --milestone "$MILESTONE"

# PR#6 子 Issues (调整为验收现有功能)
create_issue_and_record "pr6_sub1" \
  --title "验收：discover 基础能力（trace → 初始双契约）" \
  --body $'动机：验证降低门槛功能已实现。\n验收：测试覆盖；最小 spec 通过 schema 校验；局限性注明。' \
  --label "P1,area:validate,type:feat,CE" --milestone "$MILESTONE"

create_issue_and_record "pr6_sub2" \
  --title "示例：GitHub Actions minimal.yml & combo.yml" \
  --body $'动机：CI 一把梭。\n验收：公共仓库可直接跑通（含退出码断言）。' \
  --label "P1,area:release,type:docs,CE" --milestone "$MILESTONE"

create_issue_and_record "pr6_sub3" \
  --title "报告：覆盖率/摘要在 CE 构建下稳定（快照）" \
  --body $'动机：避免 Pro 代码路径泄漏。\n验收：HTML/JSON/JUnit 三格式快照一致。' \
  --label "P1,area:report,type:fix,CE" --milestone "$MILESTONE"

create_issue_and_record "pr6_sub4" \
  --title "文档：从 trace 发现双契约 & GHA 指南" \
  --body $'动机：端到端教程。\n验收：/docs/discovery/from-trace.md、/docs/ci/github-actions.md。' \
  --label "P1,area:docs,type:docs,CE" --milestone "$MILESTONE"

# Step 3: 创建跟踪 Issues 并回填子 Issue 链接
echo ""
echo "Step 3: 创建跟踪 Issues..."

# PR#1 跟踪 Issue
pr1_sub1_id=$(get_issue_id "pr1_sub1")
pr1_sub2_id=$(get_issue_id "pr1_sub2")
pr1_sub3_id=$(get_issue_id "pr1_sub3")
pr1_sub4_id=$(get_issue_id "pr1_sub4")

create_issue_and_record "pr1_tracking" \
  --title "[CE][PR#1] 退出码对齐与测试护栏（0–4 常量化）" \
  --body "目标：统一 validate 退出码为 0–4，防止 CI 漏判；引入常量；补充集成测试与文档。

**子任务**
- [ ] #$pr1_sub1_id: internal/cli/exitcode 常量化（OK=0, INPUT=1, SCHEMA=2, ASSERT=3, GATE=4）
- [ ] #$pr1_sub2_id: validate 全面使用常量并清理分歧
- [ ] #$pr1_sub3_id: CLI 集成测试覆盖各类失败码
- [ ] #$pr1_sub4_id: README 与 CLI 文档更新 0–4 表格

**验收**
- 所有 os.Exit 调用均指向 exitcode 常量
- 集成测试断言门控失败返回 4，其它路径不回归
- 文档与实现一致" \
  --label "P0,area:cli,type:fix,CE" --milestone "$MILESTONE"

# PR#2 跟踪 Issue
pr2_sub1_id=$(get_issue_id "pr2_sub1")
pr2_sub2_id=$(get_issue_id "pr2_sub2")
pr2_sub3_id=$(get_issue_id "pr2_sub3")
pr2_sub4_id=$(get_issue_id "pr2_sub4")
pr2_sub5_id=$(get_issue_id "pr2_sub5")

create_issue_and_record "pr2_tracking" \
  --title "[CE][PR#2] 基线+阈值门控全链路（relative 计算+报告字段+策略）" \
  --body "目标：实现\"基线与阈值（基础）\"。

**子任务**
- [ ] #$pr2_sub1_id: CLI 将 --baseline 贯通到 EvaluateGate
- [ ] #$pr2_sub2_id: EvaluateGate 支持 relative（(cur-v0)/v0）
- [ ] #$pr2_sub3_id: 报告新增 baseline_value/delta_abs/delta_pct（HTML/JSON/JUnit）
- [ ] #$pr2_sub4_id: 缺失基线策略 --baseline-missing={fail|treat-as-absolute}
- [ ] #$pr2_sub5_id: 文档：报告字段与示例截图

**验收**
- 有/无基线两路径稳定
- 报告展示对齐
- 单/集成测试齐全" \
  --label "P0,area:validate,area:report,type:feat,CE" --milestone "$MILESTONE"

# PR#3 跟踪 Issue
pr3_sub1_id=$(get_issue_id "pr3_sub1")
pr3_sub2_id=$(get_issue_id "pr3_sub2")
pr3_sub3_id=$(get_issue_id "pr3_sub3")
pr3_sub4_id=$(get_issue_id "pr3_sub4")

create_issue_and_record "pr3_tracking" \
  --title "[CE][PR#3] 因果/DAG 校验：偏序与边约束（容差/非重叠/父子回退）" \
  --body "目标：把存在性检查升级为真正的因果校验。

**子任务**
- [ ] #$pr3_sub1_id: 实现 CheckCausality(边约束+容差)
- [ ] #$pr3_sub2_id: validateWithCausality 接入新校验器
- [ ] #$pr3_sub3_id: CLI 增加 --causality-tolerance，默认 10–50ms
- [ ] #$pr3_sub4_id: 文档：规则/正反例/违规输出

**验收**
- 三类样例（正确链/顺序打乱/并发不重叠）通过
- OTLP 场景兼容" \
  --label "P0,area:validate,type:feat,CE" --milestone "$MILESTONE"

# PR#4 跟踪 Issue
pr4_sub1_id=$(get_issue_id "pr4_sub1")
pr4_sub2_id=$(get_issue_id "pr4_sub2")
pr4_sub3_id=$(get_issue_id "pr4_sub3")

create_issue_and_record "pr4_tracking" \
  --title "[CE][PR#4] FlowSpec Schema 对齐：graph 优先 + oneOf(flow)" \
  --body "目标：对外 schema 与内部一致，graph 可用，flow 兼容。

**子任务**
- [ ] #$pr4_sub1_id: schemas/flowspec.schema.json 增加 oneOf 与稳定 \$id
- [ ] #$pr4_sub2_id: 解析器优先 graph 失败回退 flow
- [ ] #$pr4_sub3_id: 文档：graph 默认示例 + VSCode 关联

**验收**
- graph/flow 两示例均能通过校验" \
  --label "P0,area:schema,type:feat,CE" --milestone "$MILESTONE"

# PR#5 跟踪 Issue (调整后)
pr5_sub1_id=$(get_issue_id "pr5_sub1")
pr5_sub2_id=$(get_issue_id "pr5_sub2")
pr5_sub3_id=$(get_issue_id "pr5_sub3")

create_issue_and_record "pr5_tracking" \
  --title "[CE][PR#5] CE 构建隔离与可见差异验收（build tag、version、报告角标）" \
  --body "目标：验收并完善独立 CE 产物与可见差异。

**子任务**
- [ ] #$pr5_sub1_id: 验收 //go:build ce 隔离 telemetry/org（补充 no-op stub）
- [ ] #$pr5_sub2_id: 验收 --version 输出 vX.Y.Z-ce 与 HTML 报告角标 CE
- [ ] #$pr5_sub3_id: 文档：安装三通道 & 零遥测声明

**验收**
- go build -tags ce 产物不含遥测依赖
- 报告/CLI 均显式 CE 标识
- 测试覆盖完整" \
  --label "P1,area:release,area:report,area:docs,type:feat,CE" --milestone "$MILESTONE"

# PR#6 跟踪 Issue (调整后)
pr6_sub1_id=$(get_issue_id "pr6_sub1")
pr6_sub2_id=$(get_issue_id "pr6_sub2")
pr6_sub3_id=$(get_issue_id "pr6_sub3")
pr6_sub4_id=$(get_issue_id "pr6_sub4")

create_issue_and_record "pr6_tracking" \
  --title "[CE][PR#6] 体验验收与样例（discover + GHA workflows + 报告覆盖率）" \
  --body "目标：验收装好即用体验，补充完善。

**子任务**
- [ ] #$pr6_sub1_id: 验收 discover 基础能力（trace → 初始双契约）
- [ ] #$pr6_sub2_id: examples/ci/github-actions/{minimal.yml, combo.yml}
- [ ] #$pr6_sub3_id: 报告覆盖率/摘要在 CE 构建下快照稳定
- [ ] #$pr6_sub4_id: 文档：从 trace 发现 + GHA 指南

**验收**
- 两套 workflow 复制即跑通
- discover 生成最小可校验 spec
- 测试覆盖完整" \
  --label "P1,area:validate,area:docs,area:release,type:feat,CE" --milestone "$MILESTONE"

# Step 4: 输出 Issue 映射表
echo ""
echo "========================================="
echo "Issue 创建完成！映射表："
echo "========================================="
cat "$ISSUE_MAP_FILE"

# Step 5: 生成分支和 PR 创建命令
echo ""
echo "========================================="
echo "PR 执行命令（保存以备后用）"
echo "========================================="

pr1_tracking_id=$(get_issue_id "pr1_tracking")
pr2_tracking_id=$(get_issue_id "pr2_tracking")
pr3_tracking_id=$(get_issue_id "pr3_tracking")
pr4_tracking_id=$(get_issue_id "pr4_tracking")
pr5_tracking_id=$(get_issue_id "pr5_tracking")
pr6_tracking_id=$(get_issue_id "pr6_tracking")

cat <<EOF > pr-commands.sh
#!/bin/bash
# PR 执行命令（根据需要逐个执行）

# PR#1: 退出码对齐
git checkout -b pr/ce-01-exitcodes "$DEFAULT_BRANCH"
# 实现代码...
git add -A
git commit -m "fix(cli): unify exit codes via internal/cli/exitcode (0–4)

Closes #$pr1_sub1_id
Closes #$pr1_sub2_id
Closes #$pr1_sub3_id
Closes #$pr1_sub4_id
Refs #$pr1_tracking_id"
git push -u origin pr/ce-01-exitcodes
gh pr create --title "[CE][PR#1] 退出码对齐与测试护栏（0–4）" \\
  --body "### 背景
按照 #$pr1_tracking_id 统一退出码标准

### 改动
- 新增 internal/cli/exitcode 常量包
- validate 命令改用统一常量
- 新增集成测试覆盖各失败码
- 更新 README 和文档

### 兼容性
- 行为变化：门控失败现在返回 4（原先可能是 3）
- 退出码标准化为 0-4

### 测试
- [x] 单测
- [x] 集成测试
- [x] 手动验证

### 风险 & 回滚
- 风险点：依赖退出码的 CI 脚本可能需要调整
- 回滚策略：单 PR 可直接 revert

Closes #$pr1_tracking_id" \\
  --base "$DEFAULT_BRANCH" --head "pr/ce-01-exitcodes" \\
  ${REVIEWERS:+--reviewer "$REVIEWERS"}

# PR#2: 基线门控
git checkout -b pr/ce-02-baseline-gates "$DEFAULT_BRANCH"
# 实现代码...
git add -A
git commit -m "feat(validate): baseline-aware gates (relative) + report fields

Closes #$pr2_sub1_id
Closes #$pr2_sub2_id
Closes #$pr2_sub3_id
Closes #$pr2_sub4_id
Closes #$pr2_sub5_id
Refs #$pr2_tracking_id"
git push -u origin pr/ce-02-baseline-gates
gh pr create --title "[CE][PR#2] 基线+阈值门控（relative+report）" \\
  --body "### 背景
实现 #$pr2_tracking_id 基线门控能力

### 改动
- CLI 传递 baseline 到 EvaluateGate
- 实现 relative 模式计算
- 报告新增 baseline_value/delta_abs/delta_pct
- 新增 --baseline-missing 策略
- 更新文档

### 兼容性
- 行为变化：有基线时启用相对门控
- 新增 CLI 参数：--baseline-missing

### 测试
- [x] 单测
- [x] 集成测试
- [x] 报告快照更新

Closes #$pr2_tracking_id" \\
  --base "$DEFAULT_BRANCH" --head "pr/ce-02-baseline-gates" \\
  ${REVIEWERS:+--reviewer "$REVIEWERS"}

# PR#3: 因果校验
git checkout -b pr/ce-03-causality-dag "$DEFAULT_BRANCH"
# 实现代码...
git add -A
git commit -m "feat(validate): DAG/causality edge constraints with tolerance

Closes #$pr3_sub1_id
Closes #$pr3_sub2_id
Closes #$pr3_sub3_id
Closes #$pr3_sub4_id
Refs #$pr3_tracking_id"
git push -u origin pr/ce-03-causality-dag
gh pr create --title "[CE][PR#3] 因果/DAG 校验升级（偏序+容差）" \\
  --body "### 背景
升级 #$pr3_tracking_id 因果校验能力

### 改动
- 实现 CheckCausality 边约束检查
- validateWithCausality 统一调用
- 新增 --causality-tolerance 参数
- 更新文档

### 兼容性
- 行为变化：更严格的因果校验
- 新增 CLI 参数：--causality-tolerance

### 测试
- [x] 单测
- [x] 集成测试
- [x] OTLP 场景测试

Closes #$pr3_tracking_id" \\
  --base "$DEFAULT_BRANCH" --head "pr/ce-03-causality-dag" \\
  ${REVIEWERS:+--reviewer "$REVIEWERS"}

# PR#4: Schema 对齐
git checkout -b pr/ce-04-schema-graph-oneof "$DEFAULT_BRANCH"
# 实现代码...
git add -A
git commit -m "feat(schema): graph-first with oneOf(flow); stable \$id

Closes #$pr4_sub1_id
Closes #$pr4_sub2_id
Closes #$pr4_sub3_id
Refs #$pr4_tracking_id"
git push -u origin pr/ce-04-schema-graph-oneof
gh pr create --title "[CE][PR#4] FlowSpec Schema：graph 优先 + oneOf(flow)" \\
  --body "### 背景
对齐 #$pr4_tracking_id Schema 标准

### 改动
- schemas/flowspec.schema.json 增加 oneOf
- 解析器优先 graph 回退 flow
- 更新文档

### 兼容性
- 行为变化：graph 成为默认格式
- flow 格式继续兼容

### 测试
- [x] Schema 校验测试
- [x] 双格式兼容测试

Closes #$pr4_tracking_id" \\
  --base "$DEFAULT_BRANCH" --head "pr/ce-04-schema-graph-oneof" \\
  ${REVIEWERS:+--reviewer "$REVIEWERS"}

# PR#5: CE 构建验收
git checkout -b pr/ce-05-buildtag-version-badge "$DEFAULT_BRANCH"
# 验收和补充...
git add -A
git commit -m "feat(release): verify CE build tag + version suffix; report CE badge

Closes #$pr5_sub1_id
Closes #$pr5_sub2_id
Closes #$pr5_sub3_id
Refs #$pr5_tracking_id"
git push -u origin pr/ce-05-buildtag-version-badge
gh pr create --title "[CE][PR#5] CE 构建隔离验收 & 可见差异" \\
  --body "### 背景
验收 #$pr5_tracking_id CE 构建差异化

### 改动
- 验收 build tag 隔离
- 验收 version/badge 显示
- 新增安装文档

### 兼容性
- 无破坏性变更

### 测试
- [x] 构建测试
- [x] 报告快照
- [x] 文档验证

Closes #$pr5_tracking_id" \\
  --base "$DEFAULT_BRANCH" --head "pr/ce-05-buildtag-version-badge" \\
  ${REVIEWERS:+--reviewer "$REVIEWERS"}

# PR#6: 体验验收
git checkout -b pr/ce-06-discover-gha-examples "$DEFAULT_BRANCH"
# 验收和补充...
git add -A
git commit -m "feat: verify discover + GHA examples; stable CE report snapshots

Closes #$pr6_sub1_id
Closes #$pr6_sub2_id
Closes #$pr6_sub3_id
Closes #$pr6_sub4_id
Refs #$pr6_tracking_id"
git push -u origin pr/ce-06-discover-gha-examples
gh pr create --title "[CE][PR#6] 体验验收：discover + GHA + 报告快照" \\
  --body "### 背景
验收 #$pr6_tracking_id 用户体验功能

### 改动
- 验收 discover 命令
- 新增 GHA 示例
- 稳定报告快照
- 完善文档

### 兼容性
- 无破坏性变更

### 测试
- [x] discover 功能测试
- [x] GHA workflow 测试
- [x] 报告快照验证

Closes #$pr6_tracking_id" \\
  --base "$DEFAULT_BRANCH" --head "pr/ce-06-discover-gha-examples" \\
  ${REVIEWERS:+--reviewer "$REVIEWERS"}
EOF

chmod +x pr-commands.sh

echo ""
echo "PR 命令已保存到 pr-commands.sh"
echo "请根据实际开发进度执行相应的 PR 命令"

echo ""
echo "========================================="
echo "清理临时文件..."
rm -f "$ISSUE_MAP_FILE"

echo ""
echo "✅ 设置完成！"
echo ""
echo "下一步："
echo "1. 查看创建的 Issues: gh issue list --milestone \"$MILESTONE\""
echo "2. 开始开发并执行 pr-commands.sh 中的命令"
echo "3. 按顺序 PR#1 → PR#6 逐个合并"