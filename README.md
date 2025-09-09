# ChoreoAtlas CLI

交互逻辑治理平台 - **Map. Verify. Steer** cross-service choreography.

基于"发现-规范-指导"的闭环理念，支持双规约模式（ServiceSpec 与 FlowSpec），提供 Atlas Scout（探索）、Atlas Proof（校验）、Atlas Pilot（指导）等组件。

## 🚀 快速开始

### 安装依赖

```bash
make deps
```

### 构建

```bash
make build
```

### 基础用法

```bash
# 静态校验（含 JSON Schema 验证）
./bin/flowspec lint --flow examples/flows/order-fulfillment.flowspec.yaml

# 动态验证
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --edition ce

# 生成 JSON 报告
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --report-format json --report-out report.json

# 生成 JUnit 报告 (适合 CI)
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --report-format junit --report-out report.xml

# 从 trace 探索生成 FlowSpec
./bin/flowspec discover --trace examples/traces/successful-order.trace.json --out discovered.yaml --title "探索的流程"

# CI 门禁模式
./bin/flowspec ci-gate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --edition ce

# 运行完整示例验证
make run-example

# M4 企业级功能使用示例

# HTML 报告 - 离线可用的企业级报告
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --report-format html --report-out report.html

# 基线门控 - 记录基线
./bin/flowspec baseline record --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --out baseline.json

# 基线门控 - 带阈值验证
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --baseline baseline.json --threshold-steps 0.9 --threshold-conds 0.95

# DAG 格式校验 - 支持复杂并发流程
./bin/flowspec lint --flow examples/flows/order-fulfillment-dag.flowspec.yaml
./bin/flowspec validate --flow examples/flows/order-fulfillment-dag.flowspec.yaml --trace examples/traces/dag-order-trace.json --causality temporal

# 因果校验模式
./bin/flowspec validate --flow examples/flows/order-fulfillment-dag.flowspec.yaml --trace examples/traces/dag-order-trace.json --causality strict  # 严格模式：需要父子关系
./bin/flowspec validate --flow examples/flows/order-fulfillment-dag.flowspec.yaml --trace examples/traces/dag-order-trace.json --causality temporal # 时序模式：基于时间戳
./bin/flowspec validate --flow examples/flows/order-fulfillment-dag.flowspec.yaml --trace examples/traces/dag-order-trace.json --causality off     # 关闭因果检查

# M3 企业级功能使用示例

# PII 脱敏 (Pro-Privacy)
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/pii-test.trace.json --mask-policy policies/pii.yaml --edition pro-privacy

# OTLP JSON 导入 (Pro-Free+)
./bin/flowspec validate --flow examples/flows/order-fulfillment-parallel.flowspec.yaml --otlp-json examples/traces/parallel-otlp.json --edition pro-free
```

## ✨ 核心特性

### M4 企业级增强功能 (当前版本)
- **HTML 报告系统**: 离线可用的企业级报告，包含覆盖度统计、详细表格和甘特图时间轴
- **基线门控**: 可配置的步骤覆盖率(90%)和条件通过率(95%)阈值，支持CI/CD质量门禁
- **DAG 语义**: 全新图格式规约，支持因果校验(strict/temporal/off)和并发流程建模

### M3 企业级增强功能
- **PII 脱敏防护**: 5种脱敏策略 (redact/hash/keep-prefix/tokenize/null)，YAML 策略配置
- **OTLP JSON 导入**: 完整支持 OpenTelemetry 标准格式，resourceSpans/scopeSpans 解析
- **因果并发校验**: 基于父子 Span 关系的调用图分析，支持并发步骤验证
- **覆盖度报告**: 步骤/条件级统计，服务覆盖度分析，JSON/JUnit 双格式输出
- **CEL 语义校验**: 基于 Google CEL 的前置/后置条件校验

### M1 增强功能
- **JSON Schema 严格校验**: 结构化验证 FlowSpec 和 ServiceSpec 格式
- **结构化报告生成**: 支持 JSON 和 JUnit XML 格式，无缝 CI 集成
- **探索式规约生成**: 从 trace.json 半自动生成 FlowSpec 雏形
- **严格时序校验**: 基于时间戳的步骤顺序验证

### MVP 基础功能
- **静态校验 (Lint)**: 验证 FlowSpec 自洽性、服务引用合法性、变量依赖连贯性
- **动态验证 (Validate)**: 将 FlowSpec 与实际执行追踪进行匹配验证
- **版本分层**: 支持 CE、Pro-Free、Pro-Privacy、Cloud 等不同功能级别
- **CI 集成**: 提供非零退出码以支持 CI/CD 流水线集成

## 业务示例

项目包含完整的"下单-扣库存-发货"电商流程示例：

- `examples/flows/order-fulfillment.flowspec.yaml` - 传统流程规约格式
- `examples/flows/order-fulfillment-dag.flowspec.yaml` - DAG图格式规约
- `examples/services/` - 各服务契约规约
- `examples/traces/` - 成功和失败场景的追踪数据

## 🎯 M4 新功能详解

### HTML 报告系统
生成离线可用的企业级HTML报告，包含：
- **Summary区**: 覆盖率统计（steps=5, covered=5(100%), condPass=96% 等）
- **详细表格**: 每步状态与断言详情（含 SKIP 原因）  
- **甘特图时间轴**: 按 Start/EndNanos 展示执行时序
- **基线门控结果**: 实时显示阈值检查状态

```bash
# 生成HTML报告
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml \
  --trace examples/traces/successful-order.trace.json \
  --report-format html --report-out report.html \
  --baseline baseline.json --threshold-steps 0.9 --threshold-conds 0.95
```

### 基线门控系统
支持质量阈值管控，确保代码质量：

```bash
# 1. 记录基线（通常在主分支执行）
./bin/flowspec baseline record \
  --flow examples/flows/order-fulfillment.flowspec.yaml \
  --trace examples/traces/successful-order.trace.json \
  --out baseline.json

# 2. 在PR/分支中验证是否达标
./bin/flowspec validate \
  --flow examples/flows/order-fulfillment.flowspec.yaml \
  --trace examples/traces/test.trace.json \
  --baseline baseline.json \
  --threshold-steps 0.9    # 90% 步骤覆盖率
  --threshold-conds 0.95   # 95% 条件通过率
  --skip-as-fail          # 将SKIP条件视为FAIL

# 返回不同退出码：
# 0: 全部通过
# 3: 验证失败  
# 4: 门控失败
```

### DAG 语义格式
支持复杂并发流程建模，替代传统线性flow格式：

```yaml
# 传统 flow 格式
flow:
  - step: "创建订单"
    call: "orderService.createOrder"
  - step: "库存扣减" 
    call: "inventoryService.reserveInventory"

# 新 DAG 格式 - 支持并发和复杂依赖关系
graph:
  nodes:
    - id: createOrder
      call: orderService.createOrder
      input:
        customerId: "${customerId}"
      output:
        orderResponse: response.body
    - id: reserveInventory
      call: inventoryService.reserveInventory
      input:
        orderId: "${orderResponse.orderId}"
    - id: checkRisk  # 与库存扣减并发执行
      call: riskService.check
      input:
        customerId: "${customerId}"
  edges:
    - from: createOrder
      to: reserveInventory
    - from: createOrder  
      to: checkRisk        # 并发分支
    - from: reserveInventory
      to: processPayment
    - from: checkRisk
      to: processPayment   # 汇聚点
```

**DAG 校验规则**：
- ✅ 无环检测（循环依赖）
- ✅ 连通性验证（所有节点可达）
- ✅ 变量流向分析（确保变量在使用前已定义）
- ✅ 因果关系校验（三种模式）

**因果校验模式**：
- `strict`: 基于OTLP parent-child span关系的严格验证
- `temporal`: 基于时间戳的时序验证（默认）
- `off`: 关闭因果检查，仅做宽松匹配

## 🔧 CI/CD 集成

### GitHub Actions 自动化
项目内置完整的CI/CD流水线，支持M4企业级质量门禁：

```yaml
# 自动执行的验证流程
✅ 代码质量检查 (lint + test)
✅ 多格式报告生成 (JSON + JUnit + HTML)  
✅ DAG格式验证 (成功/失败场景)
✅ 基线门控检查 (90% + 95% 阈值)
✅ 因果关系验证 (三种模式)
✅ 企业功能测试 (OTLP + PII)
```

### 质量门禁配置
PR会自动触发严格的质量检查：

```bash
# 质量门禁标准
Step Coverage:    ≥ 90%    # 步骤覆盖率
Condition Pass:   ≥ 95%    # 条件通过率  
Semantic Check:   启用      # 语义校验
Exit Codes:       0=pass, 3=validation-fail, 4=gate-fail
```

### 报告产物上传
每次CI运行都会生成并上传：
- `report.html` - 离线可用的企业级HTML报告
- `report.junit.xml` - JUnit格式，CI工具可直接解析  
- `report.json` - 结构化JSON数据，便于后续处理
- `baseline.json` - 基线数据，用于质量对比
- `quality-gate-report.html` - 质量门禁详细报告

### 本地CI测试
```bash
# 模拟CI环境测试
make build
make test  
make lint

# 测试质量门禁
./bin/flowspec baseline record --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --out local-baseline.json
./bin/flowspec validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --baseline local-baseline.json --threshold-steps 0.9 --threshold-conds 0.95
```

## 开发

```bash
# 代码格式化和检查
make lint

# 运行测试
make test

# 清理构建产物
make clean
```

## 架构

- `cmd/flowspec/` - CLI 入口点
- `internal/cli/` - 命令行处理逻辑
- `internal/spec/` - 规约加载和解析
- `internal/validate/` - 静态和动态验证逻辑，包含因果校验
- `internal/trace/` - 追踪数据处理，支持 OTLP JSON 格式
- `internal/mask/` - PII 脱敏策略引擎
- `internal/edition/` - 版本特性管理
- `policies/` - 脱敏策略配置文件

## 版本支持

| 版本 | 特性 |
|------|------|
| CE | 基础 Lint + 文件 Validate |
| Pro-Free | + OTLP 采集 |
| Pro-Privacy | + PII 脱敏 |
| Cloud | + 远端存储协作 |