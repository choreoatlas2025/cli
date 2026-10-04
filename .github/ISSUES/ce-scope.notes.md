# CE 功能范围：维护旁注

对应 [CE 功能范围](ce-scope.md)。按现有实现归类，不新增产品能力。

## 实现对应关系

| 功能域 | 操作入口 | 代码位置 |
|---|---|---|
| 契约准备 | `init`、`discover`、`spec convert` | `internal/cli/init.go`、`internal/cli/discover.go`、`internal/cli/root.go`、`internal/spec/`、`templates/` |
| 契约校验 | `lint`、`validate` | `internal/cli/lint.go`、`internal/cli/validate.go`、`internal/validate/`、`internal/schemas/` |
| 本地指标与基线 | `baseline record`；`validate` 的基线和阈值选项 | `internal/cli/baseline.go`、`internal/baseline/` |
| 结果呈现 | 校验输出、报告选项、退出码 | `internal/cli/report.go`、`internal/report/html/`、`internal/cli/exitcode/` |

`spec validate` 是静态 `lint` 的别名；`run validate` 是动态 `validate`
的入口。命令别名、表达方式和输出格式不另算功能域。

## 当前限制

- 发现生成的是初稿，输入输出映射和业务断言需要人工确认。
- CEL 编译、类型或求值错误直接判失败。跨步骤输出在单次验证内传递：Flow 按阶段发布，DAG 仅向后继提供祖先输出；并行同级不共享输出，独立前驱的同名输出拒绝合并。
- `${变量.字段}` 保留输出类型；外部初始变量未提供注入入口，读取未绑定变量时失败。`--semantic=false` 同时关闭 CEL 断言与输出映射求值。
- 因果检查依赖追踪记录中的时间戳、父子关系等字段。
- `internal/mask/` 保留，但未接入 CLI，不列为已交付的用户功能。
- `internal/trace/otlpjson.go` 保留，但 CLI 未接入；CE 范围不包含 OTLP
  接收、发送或直接导入。
- `internal/telemetry/` 为使用统计空实现；处理用户提供的 span 属性仍属于本地校验。

## 边界残留与维护约束

- `ci-gate`、`init --ci`、客户 CI 模板及集成教程仍在仓库中，但不属于当前
  确认的 CE 功能范围。仓库自身的构建、测试和发布工作流继续保留。
- 标准退出码、本地指标和基础阈值判定属于 CE；用户自行接入 CI 不构成
  CE 内置集成能力。
- 帮助中列出的 `--format`、`--log-level`、`--summary` 尚未实现，不作为支持项。
- 保留既有基础实现，依照主稿归类；清单之外的能力不自动纳入 CE。
- 本 issue 草稿不执行上述残留的代码清理。合并只能由人类执行。
