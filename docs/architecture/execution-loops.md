# CE 执行边界

本文件约束代码职责。execution loop 是有输入、实施载体、输出、验收和反馈的执行作用域；单次命令、测试或提交是内部动作。功能 loop 如何组合这些载体另行评估，不由包名推断能力已经成立。

| 执行 loop | 输入 | 实现载体 | 输出与消费方 | 验收与失败反馈 |
|---|---|---|---|---|
| E01 本地协调与初始化 | 命令参数、模板、文件路径 | `internal/cli`、`templates` | 调用其他载体，提交本地输出 | 参数错误不进入内核；生成集验收失败不覆盖已有文件 |
| E02 契约准备 | 捕获的契约文件与配置 | `internal/spec`、`internal/schemas`、`validate/plan.go`、`validate/static.go` | 契约快照、不可变编译计划 | Schema、引用、图和表达式错误归契约准备；不可伪装成运行通过 |
| E03 本地输入捕获 | 本地 trace 文件 | `internal/input`、`internal/trace` | 捕获字节、文件身份、span 与字段存在性 | 身份和时间不合格反馈输入层；缺失字段不能变成零值证据 |
| E04 调用实例匹配 | 编译契约、已捕获 span | `validate/flow_match.go`、`validate/graph_match.go`、`validate/dataflow.go` | 步骤与确切 span 实例、关系检查、变量可见域 | span 不复用；Flow/DAG 保留各自关系语义，不能自行构造请求证据 |
| E05 证据绑定与规则求值 | 匹配实例、客户表达式、可见变量 | `internal/evidence`、`validate/cel.go` | 字段来源、条件判定、成功步骤输出 | 声明与观测分离；缺失证据、规则违反、执行错误分别记录 |
| E06 本地结果与基线 | 统一步骤结果、输入身份、阈值 | `internal/verdict`、`internal/result`、`internal/baseline` | 单一最终结论、指标与本地基线比较 | 不充分或失败证据不能被阈值改判成功，不能录为成功基线 |
| E07 本地报告与输出 | 已判定结果、捕获的展示数据 | `internal/report`、`internal/fileio` | JSON、JUnit、HTML 与原子文件输出 | 各格式消费同一结果，不重新解释判定；外部字段安全展示 |
| E08 契约草案生成 | 已捕获 trace | `internal/discovery`（复用调用图及契约检查） | 带观察性质的契约草案，交 E01 写入 | 保留实例和关系；不能把样本观察升级成客户业务规则 |

## 核心不变量

- 客户契约声明属于预期，span 属性属于观测，两者通过明确比较产生判定。
- 观测字段绑定确切 span 和属性路径；文件身份由输入快照与执行记录绑定。无 trace ID 时不伪造它。
- 缺失证据、无效契约、结构不符合、规则违反、执行错误都有独立原因。对既有退出码保持失败封闭：无法证明通过就不返回成功。
- 编译计划不拥有调用间可变状态。匹配、变量作用域、证据绑定、求值分别承担自己的职责。
- `verdict` 是无领域依赖的判定记录；报告不依赖验证算法，验证算法不依赖报告或 CLI。
- 生成草案不写文件。协调层负责完整生成集的校验与写入边界。
- 仓库构建和发布 CI 属于工程交付，不能由此推导 CE 提供客户 CI 服务。

## 接口与依赖方向

| 边界 | 入口与输出 | 状态及反馈的归属 |
|---|---|---|
| E02 → E04/E05 | `CompilePlan` → 冻结的 `ContractPlan`；变更契约或配置必须重新编译 | 编译失败为 `invalid_contract`，不能绑定运行 span |
| E03 → E04 | `input.Snapshot`、`trace.Parse` → 捕获输入与 span；`SpanKey` 仅在输入范围内定位 | 没有 trace ID 时仅有文件身份；不能假称分布式身份已证明 |
| E04 → E05 | `matchFlowSteps` / `matchGraphSteps` → `matchedStep`（契约步骤、确切调用实例、结构结果） | `structure_mismatch` 留在匹配结果；匹配器不执行 CEL、不发布输出 |
| E05 → E06 | `evidence.Bind` → 观测投影与属性来源；`evaluateStep` → 步骤/条件记录及成功输出 | `missing_evidence`、`rule_violation`、`execution_error` 分开；结构未通过不求值 |
| E06 → E07/E01 | `result.New` → 独立拥有的 `result.Report`；CLI 每次只生成一份最终记录 | `verdict` 决定成功；报告、退出码与指标不能重新改判 |
| E08 → E01 | `discovery.FlowYAML` / `BuildServiceSpecFiles` → 字符串/文件字节集 | 草案资格失败交回协调层；`discoverAndPersist` / `initializeProject` 校验并提交完整生成集 |

`internal/spec/opname.go` 是 E04 与 E08 共用的操作身份协议；`internal/fileio` 是本地写入载体。共享辅助载体不等于混合各 loop 的判定责任。

`request`、`response` 是实际观测；`expected` 是契约声明经当前变量解析后的值；`vars` 是本次运行中成功步骤的输出。声明类断言不能证明实际请求；运行规则须显式比较观测和预期。字段来源记录的是投影绑定，不能当成 CEL 实际读取了全部字段的记录。原有通过 `request` 读取声明的表达式须迁移到 `expected`，不能保留静默回退。

执行错误和证据不足继续使用失败退出码，增加原因字段，不引入自动放过或新的客户 CI 服务。JSON 与 HTML 保留步骤证据，JUnit 的 testcase `system-out` 包含完整步骤记录（此前只有条件数组）。

依赖检查在 `internal/architecture/boundaries_test.go` 中：中立判定无领域依赖，输入/证据不得依赖契约或求值，发现不得拥有写入，匹配不得求值，报告和基线不得依赖验证算法。每次 `go test ./...` 都执行这些约束。

## 顺序实施与验收

每项本地检查通过并独立提交后，才进入下一项。每次提交的检查记录保存在本地 `bin/execution-foundation-20261007/`，Git 提交绑定源码版本；日志是执行证据，不能代替功能或发布准入。

| 项 | 范围 | 完成标准 |
|---|---|---|
| 1 | 核心记录 | 中立结果模型及兼容入口；原因不会被 PASS 字符串或阈值掩盖 |
| 2 | 证据绑定 | 观测与声明隔离；缺失、错误值和正确值反例；字段可反查来源 |
| 3 | 验证内核 | Flow/DAG 共享绑定与求值；结构匹配不求值、不导出变量；并发状态独立 |
| 4 | 职责整理 | 草案算法从 CLI/spec 移出；生成无写入；依赖约束有自动检查 |
| 5 | 结果消费 | 报告消费一份结果；基线使用同一成功判定；格式一致性及实际 CLI 检查 |
