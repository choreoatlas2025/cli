---
title: "CE 功能范围"
status: draft
---

# CE 功能范围

| 功能域 | 包含的现有功能 |
|---|---|
| 契约准备 | 本地初始化；ServiceSpec、FlowSpec 读取与解析；从 trace 发现契约初稿；DAG 转顺序流程 |
| 契约校验 | 结构与引用检查；调用匹配；CEL 前后置条件；时序、因果、并行及 DAG 约束检查 |
| 本地指标与基线 | 步骤覆盖率、条件通过率；基础基线记录与比较；本地阈值判定 |
| 结果呈现 | 控制台结果；HTML、JSON、JUnit 本地报告；标准退出码 |

输入为本地 ServiceSpec、FlowSpec 和原生 trace JSON（顶层 `spans` 数组）。
基线和报告保存于本地。

上述清单为 CE 的功能范围。实现对应关系和已知限制见
[维护旁注](ce-scope.notes.md)。
