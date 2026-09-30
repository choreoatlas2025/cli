# ChoreoAtlas CLI（简体中文）

[![Version](https://img.shields.io/github/v/tag/choreoatlas2025/cli?label=version)](https://github.com/choreoatlas2025/cli/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/docker/v/choreoatlas/cli?label=docker)](https://hub.docker.com/r/choreoatlas/cli)

以契约即代码（Contract-as-Code）治理跨服务编排：发现、校验、引导（Discover → Specify → Guide）。

Community Edition（CE）版本：零遥测，完全离线可用。

- Atlas Scout：从追踪数据发现/生成契约
- Atlas Proof：将编排与真实运行时行为进行对照验证
- Atlas Pilot：静态检查（Lint）与建模指引

English? See README.md

## 🚀 快速开始

建议先用 Git、Docker 和 Make 运行已验证的通过/失败对照示例：

```bash
git clone https://github.com/choreoatlas2025/quickstart-demo.git
cd quickstart-demo
make demo
```

打开 `reports/successful-order-report.html` 与 `reports/failed-payment-report.html`，也可以先在线查看[通过报告](https://cq365.eu.org/reports/successful-order-report.html)和[失败报告](https://cq365.eu.org/reports/failed-payment-report.html)。前者的 5 个流程步骤通过，后者的支付失败触发 Gate 失败。示例追踪缺少部分输入字段，因此部分前置条件显示 `SKIP`；请先阅读[快速开始指南](https://cq365.eu.org/docs/zh/guide/getting-started)。

#### Docker（推荐）
```bash
# 拉取并查看帮助
docker run --rm choreoatlas/cli:latest --help

# 挂载当前目录进行校验
docker run --rm -v $(pwd):/workspace choreoatlas/cli:latest lint --flow /workspace/your.flowspec.yaml
```

#### Homebrew（即将发布）
```bash
brew tap choreoatlas2025/tap
brew install choreoatlas
```

#### 手动下载
从 [Releases](https://github.com/choreoatlas2025/cli/releases) 下载与你平台匹配的二进制，添加到 PATH。

常用别名：
```bash
alias ca=choreoatlas
```

### 基本用法

```bash
# 交互式生成入门目录
ca init

# 静态校验（包含 JSON Schema 验证）
ca lint --flow examples/flows/order-fulfillment.flowspec.yaml

# 基于 trace 的动态校验（默认启用语义校验与“时间因果”模式）
ca validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json

# 生成 JSON/JUnit/HTML 报告
ca validate --flow examples/flows/order-fulfillment.flowspec.yaml \
  --trace examples/traces/successful-order.trace.json \
  --report-format html --report-out report.html

# 从 trace 发现并生成双契约（FlowSpec + ServiceSpec）
ca discover --trace examples/traces/successful-order.trace.json \
  --out discovered.flowspec.yaml \
  --out-services ./services
  # discover 默认开启 JSON Schema + Lint 门禁；失败则不落盘
  # 如需跳过（不推荐）：加 --no-validate
  # 说明：discover 默认不生成 input，遥测属性（http.*、otel.* 等）将自动转换为 ServiceSpec 的前/后置条件；
  # 如需调用实参，请在生成后按需补齐 path/query/headers/body。

# CI Gate（组合 lint + validate，并提供标准退出码）
ca ci-gate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json
```

## 🧾 TL;DR 速查表

```bash
# 一键生成入门示例（FlowSpec + ServiceSpec + Trace）
ca init

# Lint 当前目录下的 FlowSpec（默认读取 .flowspec.yaml）
ca lint --flow .flowspec.yaml

# 用 trace 做动态校验（时间因果）
ca validate --flow .flowspec.yaml --trace trace.json

# 严格门禁：100% 步骤覆盖 + 100% 条件通过
ca validate --flow .flowspec.yaml --trace trace.json \
  --threshold-steps 1.0 --threshold-conds 1.0 --skip-as-fail

# 记录一次基线
ca baseline record --flow .flowspec.yaml --trace trace.json --out baseline.json

# 使用基线+阈值做闸门；基线缺失时退化为绝对阈值模式
ca validate --flow .flowspec.yaml --trace trace.json \
  --baseline ci/baseline.json --baseline-missing treat-as-absolute \
  --threshold-steps 0.9 --threshold-conds 0.95

# 产出 HTML 报告（同样支持 json|junit）
ca validate --flow .flowspec.yaml --trace trace.json \
  --report-format html --report-out report.html

# 从 trace 生成双契约
ca discover --trace trace.json --out discovered.flowspec.yaml --out-services ./services

# 批量校验一个目录下的所有 trace
for f in traces/*.json; do ca validate --flow .flowspec.yaml --trace "$f"; done
```

## ✨ 核心能力

- 双契约：FlowSpec（编排）+ ServiceSpec（服务操作契约）
- 静态校验（lint）与动态校验（validate）
- 多格式报告：JSON / JUnit / HTML（CE 标识）
- 基线录制与闸门（Coverage、条件通过率阈值）
- CI/CD 集成（标准化退出码）

## 📋 契约结构示例

### FlowSpec（推荐使用 Graph/DAG 格式）
```yaml
info:
  title: "Order Fulfillment Process"
  version: "1.0.0"
services:
  orderService:
    spec: "./services/order-service.servicespec.yaml"
  inventoryService:
    spec: "./services/inventory-service.servicespec.yaml"
graph:
  nodes:
    - id: "createOrder"
      call: "orderService.createOrder"
      output:
        orderId: "response.orderId"
    - id: "checkInventory"
      call: "inventoryService.reserveInventory"
      depends: ["createOrder"]
      input:
        orderId: "${orderId}"
      output:
        reservationId: "response.reservationId"
```

### FlowSpec（顺序式 Legacy 格式）
```yaml
info:
  title: "Order Fulfillment Process"
services:
  orderService:
    spec: "./services/order-service.servicespec.yaml"
flow:
  - step: "Create Order"
    call: "orderService.createOrder"
    output:
      orderId: "response.orderId"
  - step: "Reserve Inventory"
    call: "inventoryService.reserveInventory"
    input:
      orderId: "${orderId}"
```

### ServiceSpec
```yaml
service: "OrderService"
version: "1.0.0"
operations:
  - operationId: "createOrder"
    description: "Create a new order"
    preconditions:
      "validCustomer": "has(input.customerId) && input.customerId != ''"
      "hasItems": "size(input.items) > 0"
    postconditions:
      "orderCreated": "has(response.body.orderId)"
      "statusOk": "response.status == 200"
```

## 🧰 CLI 参考

```text
choreoatlas init
  --mode string          初始化模式：template|trace
  --trace string         指定 trace.json（from-trace 模式）
  --ci string            GitHub Actions 模板：none|minimal|combo
  --examples             复制 examples/* 示例目录
  --yes                  默认接受交互提示
  --force                覆盖已存在文件
  --out string           目标目录（默认 "."）
  --title string         自定义 FlowSpec 标题

choreoatlas lint
  --flow string          FlowSpec 文件路径（默认 ".flowspec.yaml"）
  --schema               是否启用 JSON Schema 严格校验（默认 true）

choreoatlas validate
  --flow string          FlowSpec 文件路径（默认 ".flowspec.yaml"）
  --trace string         trace.json 路径（必需）
  --semantic bool        语义校验（CEL），默认启用
  --causality string     因果模式：strict|temporal|off（默认 "temporal"）
  --causality-tolerance int  因果容差（毫秒，默认 50）
  --baseline string      基线文件
  --baseline-missing string  基线缺失策略：fail|treat-as-absolute（默认 "fail"）
  --threshold-steps float    步骤覆盖阈值（默认 0.9）
  --threshold-conds float    条件通过率阈值（默认 0.95）
  --skip-as-fail        将 SKIP 视为 FAIL
  --report-format string 报告格式：json|junit|html（可选）
  --report-out string    报告输出路径（与 --report-format 一起使用）

choreoatlas discover
  --trace string         trace.json 路径（必需）
  --out string           FlowSpec 输出（默认 "discovered.flowspec.yaml"）
  --out-services string  ServiceSpec 输出目录（默认 "./services"）
  --title string         FlowSpec 标题
  --no-validate          跳过 Schema+Lint 门禁（不推荐）
  # 说明：discover 默认不生成 input；遥测属性会转换为 ServiceSpec 断言。

choreoatlas ci-gate
  --flow string          FlowSpec 文件路径
  --trace string         trace.json 路径

choreoatlas baseline record
  --flow string          FlowSpec 文件路径（默认 ".flowspec.yaml"）
  --trace string         trace.json 路径（必需）
  --out string           基线输出文件（默认 "baseline.json"）
```

说明：
- 未显式指定时，默认读取当前目录下 `.flowspec.yaml`。
- `services.*.spec` 为相对 FlowSpec 的相对路径。
- 推荐使用 Graph(DAG) 格式；顺序式 `flow:` 仍受支持。

## 🧪 Trace 输入格式

CE 读取一个简单 JSON 文件，结构如下：

```json
{
  "spans": [
    {
      "name": "createOrder",
      "service": "orderService",
      "startNanos": 1693910000000000000,
      "endNanos": 1693910000100000000,
      "attributes": {"response.status": 201}
    }
  ]
}
```

默认采用“时间因果（temporal）”模式。若 attributes 含 OTLP 风格的 `otlp.parent_span_id` 与 `otlp.span_id`，可切换为 `--causality strict` 利用父子关系进行更严格的因果验证。

## 🧩 典型工作流

1) 从 trace 生成 → 人工细化 → 校验
```bash
ca discover --trace traces/happy.json --out flow.flowspec.yaml --out-services ./services
# 编辑/细化生成的 FlowSpec 与 ServiceSpec
ca lint --flow flow.flowspec.yaml
ca validate --flow flow.flowspec.yaml --trace traces/happy.json \
  --report-format html --report-out report.html
```

2) 录制基线并设置闸门
```bash
ca baseline record --flow flow.flowspec.yaml --trace traces/happy.json --out ci/baseline.json
ca validate --flow flow.flowspec.yaml --trace traces/regression.json \
  --baseline ci/baseline.json --threshold-steps 0.9 --threshold-conds 0.95
```

3) 批量校验
```bash
for f in traces/*.json; do ca validate --flow flow.flowspec.yaml --trace "$f"; done
```

## 🔧 CI/CD 集成

### GitHub Actions 示例
```yaml
name: ChoreoAtlas Validate
on: [push, pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run validation (CI gate)
        run: |
          choreoatlas ci-gate \
            --flow specs/main-flow.flowspec.yaml \
            --trace traces/integration-test.trace.json
      - name: Generate reports
        run: |
          choreoatlas validate \
            --flow specs/main-flow.flowspec.yaml \
            --trace traces/integration-test.trace.json \
            --report-format junit --report-out junit.xml
          choreoatlas validate \
            --flow specs/main-flow.flowspec.yaml \
            --trace traces/integration-test.trace.json \
            --report-format html --report-out report.html
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: choreoatlas-reports
          path: |
            junit.xml
            report.html
```

### 退出码
- `0`：全部通过
- `1`：通用 CLI 错误（参数无效等）
- `2`：文件不存在或解析错误
- `3`：校验失败（规范与 trace 不匹配）
- `4`：闸门失败（阈值/基线不满足）

### 报告格式
- JSON：结构化数据，便于程序消费
- JUnit XML：可直接集成 CI 系统
- HTML：可视化时间线 + 概览（带 CE 徽标）

## 🧱 故障排查

- “flowspec cannot have both 'graph' and 'flow' fields” → 二选一。
- “no matching span found in trace” → 检查 `service.operation` 是否与 FlowSpec 一致；确认因果模式与顺序是否匹配。
- “DAG structure validation failed” → 修复图中的环、缺失节点、不可达节点等。
- 基线缺失 → 使用 `--baseline-missing treat-as-absolute` 仅按阈值判定。
- ServiceSpec 相对路径 → 以 FlowSpec 文件所在目录为基准解析。

## 🏗️ 本地开发

```bash
# 下载安装依赖
go mod download

# 构建
make build

# 测试
make test

# 代码静态检查
make lint

# 清理
make clean
```

### 目录结构
```
.
├── cmd/choreoatlas/          # CLI 入口
├── internal/
│   ├── cli/                  # 命令行解析与子命令实现
│   ├── spec/                 # 规范装载/解析与生成
│   ├── validate/             # 静态与动态校验逻辑（含因果/并发）
│   ├── trace/                # Trace 输入适配
│   └── report/               # 报告生成（JSON/JUnit/HTML）
├── examples/                 # 示例
│   ├── flows/                # FlowSpec 示例
│   ├── services/             # ServiceSpec 示例
│   └── traces/               # Trace 示例
└── schemas/                  # JSON Schema 定义
```

## 🔒 CE 说明

- 零遥测（No telemetry）：不收集任何使用数据
- 离线运行：无需网络即可工作
- 可验证：`strings choreoatlas | grep telemetry` 应无匹配

## 📄 许可证

Apache 2.0，详见 [LICENSE](LICENSE)。

## 🔗 链接

- GitHub：https://github.com/choreoatlas2025/cli
- Releases：https://github.com/choreoatlas2025/cli/releases
- Docker Hub：https://hub.docker.com/r/choreoatlas/cli
- Issues：https://github.com/choreoatlas2025/cli/issues
- Discussions：https://github.com/choreoatlas2025/cli/discussions

—— ChoreoAtlas CLI：以契约即代码映射、校验并引导你的服务编排
