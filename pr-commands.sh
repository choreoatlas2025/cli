#!/bin/bash

# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0
# PR 执行命令（根据需要逐个执行）

# PR#1: 退出码对齐
git checkout -b pr/ce-01-exitcodes "main"
# 实现代码...
git add -A
git commit -m "fix(cli): unify exit codes via internal/cli/exitcode (0–4)

Closes #2
Closes #3
Closes #4
Closes #5
Refs #25"
git push -u origin pr/ce-01-exitcodes
gh pr create --title "[CE][PR#1] 退出码对齐与测试护栏（0–4）" \
  --body "### 背景
按照 #25 统一退出码标准

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

Closes #25" \
  --base "main" --head "pr/ce-01-exitcodes" \
  

# PR#2: 基线门控
git checkout -b pr/ce-02-baseline-gates "main"
# 实现代码...
git add -A
git commit -m "feat(validate): baseline-aware gates (relative) + report fields

Closes #6
Closes #7
Closes #8
Closes #9
Closes #10
Refs #26"
git push -u origin pr/ce-02-baseline-gates
gh pr create --title "[CE][PR#2] 基线+阈值门控（relative+report）" \
  --body "### 背景
实现 #26 基线门控能力

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

Closes #26" \
  --base "main" --head "pr/ce-02-baseline-gates" \
  

# PR#3: 因果校验
git checkout -b pr/ce-03-causality-dag "main"
# 实现代码...
git add -A
git commit -m "feat(validate): DAG/causality edge constraints with tolerance

Closes #11
Closes #12
Closes #13
Closes #14
Refs #27"
git push -u origin pr/ce-03-causality-dag
gh pr create --title "[CE][PR#3] 因果/DAG 校验升级（偏序+容差）" \
  --body "### 背景
升级 #27 因果校验能力

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

Closes #27" \
  --base "main" --head "pr/ce-03-causality-dag" \
  

# PR#4: Schema 对齐
git checkout -b pr/ce-04-schema-graph-oneof "main"
# 实现代码...
git add -A
git commit -m "feat(schema): graph-first with oneOf(flow); stable $id

Closes #15
Closes #16
Closes #17
Refs #28"
git push -u origin pr/ce-04-schema-graph-oneof
gh pr create --title "[CE][PR#4] FlowSpec Schema：graph 优先 + oneOf(flow)" \
  --body "### 背景
对齐 #28 Schema 标准

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

Closes #28" \
  --base "main" --head "pr/ce-04-schema-graph-oneof" \
  

# PR#5: CE 构建验收
git checkout -b pr/ce-05-buildtag-version-badge "main"
# 验收和补充...
git add -A
git commit -m "feat(release): verify CE build tag + version suffix; report CE badge

Closes #18
Closes #19
Closes #20
Refs #29"
git push -u origin pr/ce-05-buildtag-version-badge
gh pr create --title "[CE][PR#5] CE 构建隔离验收 & 可见差异" \
  --body "### 背景
验收 #29 CE 构建差异化

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

Closes #29" \
  --base "main" --head "pr/ce-05-buildtag-version-badge" \
  

# PR#6: 体验验收
git checkout -b pr/ce-06-discover-gha-examples "main"
# 验收和补充...
git add -A
git commit -m "feat: verify discover + GHA examples; stable CE report snapshots

Closes #21
Closes #22
Closes #23
Closes #24
Refs #30"
git push -u origin pr/ce-06-discover-gha-examples
gh pr create --title "[CE][PR#6] 体验验收：discover + GHA + 报告快照" \
  --body "### 背景
验收 #30 用户体验功能

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

Closes #30" \
  --base "main" --head "pr/ce-06-discover-gha-examples" \
  
