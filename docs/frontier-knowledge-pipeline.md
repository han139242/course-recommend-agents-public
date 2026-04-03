# 前沿知识更新链路设计

本文件定义“课内知识点 + 前沿论文/行业知识”的第一版数据模型与接入方式。

## 目标

- 支持围绕某个课内知识点生成课后扩展内容。
- 支持后续从论文题录、摘要、关键词中持续更新前沿知识。
- 与现有 `KnowledgeNode` 分域管理，避免直接污染基础课程图谱。

## 第一版范围

- 采集对象：论文题录、摘要、关键词、作者、会议/期刊、年份。
- 不做复杂全文理解，不直接引入大模型依赖。
- 重点实现：
  - `Paper`
  - `ResearchTopic`
  - `FrontierDigest`
  - 与 `KnowledgeNode` 的 `EXTENDS_KNOWLEDGE`、`CURATED_FOR` 连接

## 数据模型

详见：

- `pipelines/txt2neo4j/config/frontier_ontology.json`

## 在线产品的最小接入方式

- 后端提供 `GET /api/frontier/briefing?knowledge={name}`。
- 前端知识点详情页展示：
  - 对应知识点的前沿主题
  - 代表论文
  - 为什么它和课内知识相关

## 第二阶段演进

- 接入论文源抓取任务（如 arXiv / 顶会官网 / OpenAlex 等）
- 摘要清洗与主题抽取
- 自动生成课后扩展摘要
- 周期性更新 Neo4j 中的前沿知识域
