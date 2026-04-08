# 前沿知识更新链路设计

本文文件定义“课内知识点 + 前沿论文/行业知识”的第一版数据模型与接入方式。

在阅读当前仓库后，需要先明确一个边界：现阶段真正已经跑通的离线流水线主要是 `pipelines/txt2neo4j` 的课程知识图谱抽取与导入链路；前沿知识链路目前仍以本体设计、数据模型定义与在线接入预留为主，尚未形成完整可执行闭环。

## 当前状态

### 已经存在的能力

- 已有课程知识图谱离线流水线：原始 TXT -> 文本清洗 -> 结构解析 -> 知识点抽取 -> 关系抽取 -> 质量报告 -> 导入 Neo4j。
- 已有前沿知识本体定义：`pipelines/txt2neo4j/config/frontier_ontology.json`。
- 已有 Go 在线服务基础设施：认证、教师端基础管理、MySQL / Redis / Neo4j 装配。

### 尚未形成闭环的能力

- 论文元数据抓取。
- 摘要清洗与主题抽取。
- `ResearchTopic` 到课程 `KnowledgeNode` 的稳定映射。
- `FrontierDigest` 生成与缓存。
- Go 在线接口 `GET /api/frontier/briefing?knowledgeId={id}` 的真实落地。

因此，本文后续内容需要同时区分“已实现事实”和“目标设计”，避免把规划态误写成现状。

## 目标

- 支持围绕某个课内知识点生成课后扩展内容。
- 支持后续从论文题录、摘要、关键词中持续更新前沿知识。
- 与现有 `KnowledgeNode` 分域管理，避免直接污染基础课程图谱。
- 为知识点详情页提供稳定的前沿摘要、代表论文与关联理由。

## 第一版范围

- 采集对象：论文题录、摘要、关键词、作者、会议/期刊、年份。
- 不做复杂全文理解，不直接引入大模型在线依赖。
- 重点实现：
  - `Paper`
  - `ResearchTopic`
  - `FrontierDigest`
  - 与 `KnowledgeNode` 的 `EXTENDS_KNOWLEDGE`、`CURATED_FOR` 连接

## 已实现链路与目标链路的关系

### 1. 当前已实现链路：课程知识图谱流水线

当前仓库里最完整的流水线是课程知识图谱链路，核心阶段可以概括为：

1. 读取原始 TXT。
2. 清洗文本。
3. 解析目录与结构节点。
4. 抽取 `KnowledgeNode`。
5. 抽取节点关系。
6. 生成质量报告。
7. 导出 JSON / CSV / review 文件。
8. 导入 Neo4j。

这个链路已经证明当前项目具备“离线抽取 -> 结构化产物 -> 图数据库导入”的基础能力，因此前沿知识链路应尽量复用这套分阶段设计，而不是重新做一条完全不同的工程路径。

### 2. 前沿知识链路的目标阶段

前沿知识域建议沿用如下阶段：

1. `crawl_metadata`
2. `clean_abstract`
3. `extract_topics`
4. `link_to_course_knowledge`
5. `generate_frontier_digest`
6. `merge_into_neo4j`

其中，当前仓库只对这些阶段做了本体层定义，还没有对应的完整代码实现。

## 数据模型

详见：

- `pipelines/txt2neo4j/config/frontier_ontology.json`

建议稳定包含以下节点：

### `Paper`

- `paper_id`
- `title`
- `abstract`
- `year`
- `venue`
- `url`
- `source`
- `updated_at`

### `ResearchTopic`

- `topic_id`
- `name`
- `canonical_name`
- `keywords`
- `summary`

### `Venue`

- `venue_id`
- `name`
- `type`
- `rank`

### `Author`

- `author_id`
- `name`
- `affiliation`

### `FrontierDigest`

- `digest_id`
- `knowledge_id`
- `knowledge_name`
- `summary`
- `status`
- `generated_at`

相应关系建议为：

- `(:Paper)-[:MENTIONS_TOPIC]->(:ResearchTopic)`
- `(:Paper)-[:PUBLISHED_IN]->(:Venue)`
- `(:Paper)-[:AUTHORED_BY]->(:Author)`
- `(:ResearchTopic)-[:EXTENDS_KNOWLEDGE]->(:KnowledgeNode)`
- `(:FrontierDigest)-[:CURATED_FOR]->(:KnowledgeNode)`
- `(:FrontierDigest)-[:SUMMARIZES_PAPER]->(:Paper)`

## 每个阶段的输入输出建议

| 阶段 | 当前状态 | 输入 | 输出 |
| --- | --- | --- | --- |
| `crawl_metadata` | 未实现 | 论文源配置、主题种子、时间窗口 | 论文元数据 |
| `clean_abstract` | 未实现 | 原始摘要、关键词 | 清洗后的摘要与标准化关键词 |
| `extract_topics` | 未实现 | 清洗后的摘要文本 | `ResearchTopic` 与 Paper-Topic 关系 |
| `link_to_course_knowledge` | 未实现 | topic、课程 `KnowledgeNode` | `EXTENDS_KNOWLEDGE` 映射 |
| `generate_frontier_digest` | 未实现 | knowledge、topic、代表论文 | `FrontierDigest` |
| `merge_into_neo4j` | 未实现 frontier 版 | digest、topic、paper 数据 | Neo4j 前沿知识域节点与关系 |

## 在线产品的最小接入方式

建议目标接口为：

- `GET /api/frontier/briefing?knowledgeId={id}`（兼容 `knowledge={name}` 查询，但推荐使用 `knowledgeId`）

但这里要明确：

- 这是目标态接口，不是当前仓库里已经注册完成的真实路由。
- 当前 Go 在线服务路由仍以认证与教师端管理为主，frontier briefing 还需要后续模块补齐。

前端知识点详情页可消费的数据建议包括：

- 对应知识点的前沿主题列表。
- 代表论文列表。
- 为什么它和课内知识相关。
- 最近一次 digest 生成时间。
- 内容是否已就绪的状态字段。

建议响应最小结构：

```json
{
  "knowledgeId": "gradient-descent",
  "status": "ready",
  "digestSummary": "梯度下降近期在大模型训练稳定性与自适应优化方向上仍是核心基础。",
  "topics": [
    {
      "topicId": "adaptive-optimization",
      "name": "自适应优化",
      "reason": "该主题直接扩展了梯度下降在训练策略上的应用边界。"
    }
  ],
  "papers": [
    {
      "paperId": "paper_001",
      "title": "A Survey on Adaptive Optimization",
      "year": 2025,
      "venue": "NeurIPS",
      "url": "https://example.com/paper"
    }
  ],
  "generatedAt": "2026-04-08T00:00:00Z"
}
```

## 推荐的数据落点

### Neo4j

用于存储：

- `Paper`
- `ResearchTopic`
- `Venue`
- `Author`
- `FrontierDigest`
- 它们与课程 `KnowledgeNode` 的关系

### MySQL

建议二期补充：

- 抓取批次表。
- digest 缓存表。
- topic 与 knowledge 映射审计表。

原因：

- 在线接口需要稳定缓存与状态判断。
- 批次失败、重试、审计不适合只靠图数据库表达。
- 如果没有 MySQL 侧辅助落点，后续很难定位“为什么这个知识点没有前沿摘要”这类问题。

## 质量门禁建议

课程知识图谱流水线当前已有质量报告思路，前沿知识链路建议沿用并补齐：

- topic 抽取低置信度标记。
- topic -> knowledge 映射人工复核状态。
- digest 空结果状态。
- 论文去重策略。
- 幂等导入策略。

尤其需要明确：

- `paper_id` 如何生成。
- 多源抓取时如何去重。
- `ResearchTopic.canonical_name` 如何标准化。
- digest 如何重算、覆盖、失效。

## 第二阶段演进

- 接入论文源抓取任务（如 arXiv / 顶会官网 / OpenAlex 等）。
- 摘要清洗与主题抽取。
- 自动生成课后扩展摘要。
- 周期性更新 Neo4j 中的前沿知识域。
- 增加 `GET /api/frontier/briefing` 在线查询模块与缓存策略。
- 视情况补充首页目录接口、项目推荐接口与知识点详情页前沿联动。