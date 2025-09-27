# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

此文件为 Claude Code (claude.ai/code) 在此代码库中工作时提供指导。请使用中文与我交流。

## 项目概览

**ChoreoAtlas CLI** (原 FlowSpec CLI) - "Map. Verify. Steer cross-service choreography"

这是一个"契约即代码"交互逻辑治理平台，遵循双规约原则（ServiceSpec 与 FlowSpec）进行契约验证和执行。实现了 Atlas Scout（探索）、Atlas Proof（校验）、Atlas Pilot（指导）等组件。

## 当前分支状况

### 🎯 当前工作分支
- **`ce-clean`** (当前分支) ✅ **开源发布就绪**
  - CE纯净化改造完成，移除所有企业功能
  - 模块路径: `github.com/choreoatlas2025/cli`
  - 二进制文件: `choreoatlas` (主要), `ca` (别名)
  - 发布系统: GoReleaser + Docker + GitHub Actions ✅
  - 状态: 🚀 **准备推送到开源仓库**

### 🚀 功能分支 (已同步到GitHub)
- **`main`** - 主开发分支  
- **`task10-business-execution`** - 商务执行分支
- **`m4-html-baseline-dag`** - M4企业级功能 (HTML报告、基线门控、DAG语义)
- **`m3-otlp-privacy`** - M3企业级功能 (PII脱敏、OTLP导入)  
- **`m2-semantic-validation`** - M2增强功能 (CEL语义校验)

## 核心架构

### 契约系统
- **FlowSpec** (`.flowspec.yaml`): 中心化流程规约，定义步骤序列、服务调用和步骤间变量传递
- **ServiceSpec** (`service.spec.yaml`): 每服务契约定义，包含操作规约、前置条件和后置条件
- **版本分层特性门控**: 分层功能（CE、Pro-Free、Pro-Privacy、Cloud）配合特性标志

### 主要组件（单文件：`main.go`）
- **版本管理**: 不同产品层级的特性标志系统
- **规约加载与解析**: 基于 YAML 的契约加载与验证
- **静态检查**: 验证 FlowSpec 一致性、服务引用和变量依赖
- **动态验证**: 将 FlowSpec 与执行追踪匹配（当前基于文件，可扩展至 OTLP）
- **CLI 接口**: 提供 lint、validate 和 CI 集成命令

## 常用开发命令

### 构建和运行
```bash
# 构建 ChoreoAtlas CLI
make build

# 运行 lint 命令
./bin/choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml
./bin/ca lint --flow examples/flows/order-fulfillment.flowspec.yaml  # 使用别名

# 对追踪运行验证
./bin/choreoatlas validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --edition ce

# 运行 CI 门禁（组合 lint + validate）
./bin/ca ci-gate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --edition ce

# 运行示例
make run-example

# M3企业级功能演示
make run-m3

# 代码检查和测试
make lint
make test
```

### 依赖项
- 需要 Go 1.24+
- 模块路径: `github.com/choreoatlas2025/cli`
- 主要依赖:
  - `gopkg.in/yaml.v3` - YAML 解析
  - `github.com/google/cel-go` - CEL 表达式验证
  - `github.com/santhosh-tekuri/jsonschema/v5` - JSON Schema 验证

### 品牌和账户信息
- GitHub组织: https://github.com/choreoatlas2025  
- 邮箱: choreoatlas@gmail.com
- 相关账户: Twitter, NPM, Docker 已注册
- 待办: 域名注册(.com/.io/.dev)、商标检索

## 契约规约结构

### FlowSpec 格式
```yaml
info:
  title: "流程描述"
services:
  服务别名:
    spec: "./path/to/service.spec.yaml"
flow:
  - step: "步骤名称"
    call: "服务别名.操作ID"
    input:
      字段: "${变量引用}"
    output:
      变量名: "response.path"
```

### ServiceSpec 格式 (`.servicespec.yaml`)
```yaml
service: "服务名称"
operations:
  - operationId: "操作名称"
    description: "操作描述"
    preconditions:
      "条件名称": "CEL 表达式"
    postconditions:
      "条件名称": "CEL 表达式"
```

## Atlas 组件家族
- **Atlas Scout** (`discover`) - 从trace探索生成FlowSpec
- **Atlas Proof** (`validate`) - 验证编排与实际执行匹配  
- **Atlas Pilot** (`lint`) - 静态验证和指导

## 版本功能分层

### CE 社区版 (Community Edition)
**定位**: 面向个人开发者、开源项目、小团队试用  
**价值**: 用真实追踪驱动的可执行双契约（ServiceSpec + FlowSpec），在本地与 CI 做服务级语义验证、编排级时序/因果/DAG验证

**包含功能**:
- ServiceSpec + FlowSpec 双契约解析与 JSON Schema 校验
- 验证：ServiceSpec 服务级语义验证（CEL条件） + FlowSpec 编排级时序/因果/DAG校验
- 报告：HTML/JSON/JUnit；覆盖率/摘要；基线与阈值（基础）
- 发现：由 trace 生成初始 ServiceSpec + FlowSpec 双契约（基础）
- CI 集成：GitHub Action（组合式）与最小示例

**不包含/限制**:
- 无组织级治理（多项目集中策略/RBAC/审计）
- 无高级基线（趋势对比、抖动抑制、历史回放差异）
- 无托管数据/团队协作/通知
- 零数据收集，完全本地运行

**数据与隐私**: 完全不收集任何数据，零遥测，零外呼，默认离线

### Pro Free 专业免费版
**定位**: 面向小中团队与早期付费用户  
**价格**: $19/用户/月 或 $9/服务/月

**包含功能**:
- CE 全部功能
- 高级基线：趋势比较、异常/噪声抑制、历史回放与差异报告
- 组织级策略：多仓库/多服务集中治理；私有 flowspec/策略包仓库
- 团队协作：PR 检查 App 回注丰富摘要、失败溯源链接
- 审计日志（本地文件/导出）、Webhook/通知集成（Slack/Teams/Email）
- 可选连接器：Jaeger/Tempo 数据源拉取（仅必要字段）
- 遥测：匿名、可选择同意，帮助改进产品

**数据与隐私**:
- 首次运行明确征询同意；可用 --no-telemetry 或 CHOREOATLAS_TELEMETRY=0 关闭
- 最小字段：版本、子命令、总耗时、通过/失败计数、flowspec 大小区间、覆盖率区间、CI 环境类型
- 绝不包含代码、URL、请求体、服务名、仓库 URL、用户可识别信息

### Pro Privacy 专业隐私版
**定位**: 面向受合规/保密约束的团队（金融、政企、车企、医疗等）  
**价格**: $39/用户/月 或 $19/服务/月

**包含功能**:
- 等同 Pro Free 的功能集合
- 零遥测/零网络外呼（编译期完全移除）
- 离线许可文件激活；可选私有部署说明/镜像；可复现构建说明（repro build）

**数据与隐私**:
- 编译时移除 telemetry 依赖与代码路径；二进制中不出现 HTTP/端点常量
- 提供 SBOM、签名（cosign）与校验指引

### Cloud 云端版
**定位**: 面向需要"托管工作区 + 持续发现/漂移检测 + 团队协作 + 历史留存"的组织

**包含功能**:
- Web 控制台与团队工作区
- 连接器：OTLP/Jaeger/Tempo 等，持续发现、基线漂移检测、趋势可视化
- 协作：PR 检查 App、报告托管、变更订阅、注释回注
- 组织能力：SSO、RBAC、租户/项目、审计日志、合规导出、数据保留策略
- API/SDK：自动化与集成

**数据与隐私**:
- 只接收最小必要字段；支持字段屏蔽/哈希/分级存储
- 用户可导出/删除；数据保留配置（例如 30/90/180 天可选）
- 合规：SOC2 路线（后续）、ISO 27001（长期）

### 追踪格式（当前 PoC）
```json
{
  "spans": [
    {
      "name": "操作名称",
      "service": "服务名称"
    }
  ]
}
```

## 关键实现细节

- **变量引用系统**: 使用 `${变量名}` 语法进行步骤间数据流转
- **服务调用格式**: `服务别名.操作ID` 模式
- **基于版本的特性控制**: 根据版本标志的运行时特性可用性
- **CI 集成**: 为 CI/CD 流水线集成设计的退出码（失败时非零）
- **追踪匹配**: 当前为简单的名称匹配，可扩展支持具有时序/层次结构的 OTLP

## 目录结构

项目已重构为模块化架构，目录结构如下：

```
.
├── cmd/choreoatlas/          # 主入口点
├── internal/                 # 内部包，Go惯例
│   ├── cli/                  # 命令行处理
│   ├── edition/              # 版本管理 (CE/Pro-Free/Pro-Privacy/Cloud)
│   ├── spec/                 # 规约处理 (FlowSpec/ServiceSpec 双契约)
│   ├── validate/             # 验证引擎 (静态/动态/CEL/因果)
│   ├── trace/                # 追踪数据处理 (JSON/OTLP)
│   ├── report/               # 报告生成 (HTML/JSON/JUnit)
│   ├── baseline/             # 基线管理
│   ├── mask/                 # PII遮掩功能
│   ├── telemetry/            # 遥测 (按版本build tag分离)
│   └── enterprise/           # 企业功能 (按版本build tag分离)
├── examples/                 # 示例文件
│   ├── flows/               # FlowSpec示例
│   ├── services/            # ServiceSpec示例  
│   └── traces/              # 追踪数据示例
├── schemas/                 # JSON Schema定义
├── policies/                # PII策略文件
├── bin/                     # 构建产物
└── services/                # discover命令生成的服务规约
```

### 版本隔离实现

按照task13.md设计，使用Go build tags实现编译时版本功能隔离：

- **CE版本**: `//go:build ce` - 无遥测、无企业功能
- **Pro-Free版**: `//go:build profree` - 含遥测、含企业功能
- **Pro-Privacy版**: `//go:build proprivacy` - 无遥测、含企业功能  
- **Cloud版**: 所有功能，通过运行时配置控制

### 构建目标

```bash
make build-ce          # 社区版
make build-profree     # 专业免费版
make build-proprivacy  # 专业隐私版
make build-cloud       # 云端版
make build-all         # 所有版本
```