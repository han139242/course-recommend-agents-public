# 学习路径推荐优化 Auto Research 汇总

生成时间：2026-05-25

本轮调研围绕“如何最大化优化当前课程推荐智能体的学习路径推荐能力”展开，覆盖 6 条技术路线：知识追踪、知识图谱路径推荐、强化学习、GraphRAG/KG-RAG、LLM Agent、评测协议。

结构化结果目录：

- `research/learning-path-optimization/results-vpn-20260525-205653/Knowledge_Tracing_for_Learning_Path_Recommendation.json`
- `research/learning-path-optimization/results-vpn-20260525-205653/Knowledge_Graph_based_Learning_Path_Recommendation.json`
- `research/learning-path-optimization/results-vpn-20260525-205653/Reinforcement_Learning_for_Learning_Path_Recommendation.json`
- `research/learning-path-optimization/results-vpn-20260525-205653/GraphRAG_and_KG_RAG_for_Education.json`
- `research/learning-path-optimization/results-vpn-20260525-205653/LLM_Agents_for_Personalized_Tutoring_and_Recommendation.json`
- `research/learning-path-optimization/results-vpn-20260525-205653/Evaluation_Protocols_and_Benchmarks_for_Learning_Path_Recommendation.json`

## 一、结论

最推荐的路线不是“直接上大模型推荐”，而是：

> 知识追踪 KT 负责学生状态估计，知识图谱 KG 负责先修约束和路径可解释，图谱约束打分/上下文 Bandit 负责下一步选择，GraphRAG 和 LLM Agent 负责证据解释与交互闭环。

建议项目方法命名：

> KT-KGPath：融合知识追踪与跨课程知识图谱约束的个性化学习路径推荐方法

或者面向论文：

> Frontier-aware KT-KGPath: Knowledge-Tracing and Cross-Course Knowledge-Graph Enhanced Learning Path Recommendation

## 二、技术路线优先级

| 优先级 | 技术路线 | 是否建议作为核心 | 原因 |
|---|---|---|---|
| 1 | 知识追踪 + 知识图谱约束路径推荐 | 是 | 最贴当前项目，能直接利用答题记录、掌握度、Neo4j 图谱和推荐模块 |
| 2 | 上下文 Bandit / 图谱约束轻量 RL | 是，第二阶段 | 比 DQN 更容易落地，可作为“路径优化器”增强论文算法性 |
| 3 | GraphRAG / KG-RAG | 辅助核心 | 适合增强推荐解释、前沿论文证据、减少 LLM 幻觉 |
| 4 | LLM 多智能体规划 | 辅助 | 适合交互层和反思检查，不建议让 LLM 裸推荐 |
| 5 | 完整 DQN / Offline RL | 后续增强 | 论文价值高，但需要日志、奖励设计和离线评测，当前直接上风险较大 |
| 6 | 仅截图自测 | 不能作为核心 | 只能证明系统可用，不能证明推荐算法有效 |

## 三、推荐算法设计

当前项目建议先实现一个可解释打分器：

```text
Score(k) =
  alpha * Weakness(k)
+ beta  * PrerequisiteReadiness(k)
+ gamma * TargetRelevance(k)
+ delta * ForgettingRisk(k)
+ eta   * CrossCourseTransfer(k)
+ theta * FrontierRelevance(k)
- lambda * MasteredPenalty(k)
```

其中：

- `Weakness(k)`：学生在知识点 `k` 上的薄弱程度，由 KT 或掌握度估计得到。
- `PrerequisiteReadiness(k)`：学习 `k` 所需先修知识是否已满足。
- `TargetRelevance(k)`：`k` 与目标知识点/目标能力之间的图谱距离或路径贡献。
- `ForgettingRisk(k)`：根据上次学习时间和掌握度估计复习需求。
- `CrossCourseTransfer(k)`：跨课程迁移价值，例如线性代数知识对深度学习知识点的支撑。
- `FrontierRelevance(k)`：与前沿论文、热门技术主题的关联度。
- `MasteredPenalty(k)`：避免反复推荐已掌握知识点。

第一版不要直接做复杂 DQN。建议先做“图谱候选召回 + KT 状态估计 + 可解释打分排序”，实验稳定后再加 Bandit 或 DQN。

## 四、落地到当前项目的改造顺序

### 阶段 1：评测先行

先建立评测框架，避免盲目改推荐算法。

需要新增：

- benchmark task schema：学生历史、目标知识点、候选知识点、最大路径长度、先修图。
- baseline registry：随机、课程顺序、先修最短路、薄弱点优先、当前系统、KT-KGPath。
- metric 计算：`Hit@K`、`NDCG@K`、`MRR`、路径先修违例率、路径长度、难度平滑度、教师评分。

### 阶段 2：知识追踪 MVP

先不要训练大模型。可以从轻量版开始：

- 输入：答题记录、知识点 ID、正确/错误、时间戳、题目难度。
- 输出：学生对每个知识点的动态掌握概率。
- 第一版可用 BKT/DKT/SAKT 任一轻量模型，甚至先用规则版遗忘曲线作为 baseline。

### 阶段 3：图谱约束路径规划

利用 Neo4j：

- 从目标知识点反向找先修链。
- 从薄弱知识点找相关补弱节点。
- 当先修链断裂时，引入相似关系/相关关系作为 fallback。
- 过滤掉先修未满足、难度跳跃过大、已掌握度过高的节点。

### 阶段 4：上下文 Bandit / 轻量 RL

当有足够日志后，再把推荐排序升级为 Bandit：

- 状态：KT 掌握度、目标、历史路径、答题表现。
- 动作：候选知识点或资源。
- 奖励：答题提升、完成学习、采纳推荐、路径不违背先修。
- 安全约束：先修未满足的动作直接 mask。

### 阶段 5：GraphRAG 和 Agent

GraphRAG 不做主推荐器，而做：

- 推荐理由生成。
- 前沿论文证据召回。
- 知识点详情解释。
- LLM 输出路径的先修顺序校验。

LLM Agent 建议拆成三个工具调用角色：

- Learner Analyzer：读取掌握度和薄弱点。
- Path Planner：调用推荐器和 Neo4j。
- Reflector：检查先修顺序、认知负荷、解释一致性。

## 五、论文路线

推荐论文题目：

> 融合知识追踪与跨课程知识图谱约束的个性化学习路径推荐方法

英文题目：

> KT-KGPath: Knowledge-Tracing and Cross-Course Knowledge-Graph Constrained Learning Path Recommendation

核心贡献：

1. 构建跨课程 AI 知识图谱，统一表示先修、相似、应用、使用、子主题等关系。
2. 提出 KT-KGPath 推荐方法，将学生动态掌握状态与图谱先修约束结合。
3. 引入遗忘风险、跨课程迁移收益和前沿关联度，提升学习路径的个性化和前沿性。
4. 构建可复现实验协议，比较课程顺序、图谱最短路、薄弱点优先、KT-only、KG-only、KT-KGPath 等方法。

必须做的实验：

- 推荐排序实验：`Hit@K`、`NDCG@K`、`MRR`。
- 路径质量实验：先修违例率、目标覆盖率、路径冗余率、难度平滑度。
- 消融实验：去掉 KT、去掉图谱约束、去掉遗忘因子、去掉跨课程迁移、去掉前沿关联。
- 教师盲评：路径合理性、解释可信度、学习负荷。
- 系统实验：延迟、自动化任务成功率、截图自测。

## 六、专利路线

最适合先写一件核心发明专利：

> 一种融合知识追踪与跨课程知识图谱约束的个性化学习路径推荐方法、系统、设备及存储介质

主要权利要求可围绕：

1. 课程知识图谱构建与关系归一化。
2. 学生知识状态动态估计。
3. 图谱约束候选路径召回。
4. 综合薄弱度、先修满足度、遗忘风险、跨课程迁移收益和前沿关联度的路径排序。
5. 基于智能体的推荐解释与路径校验。

后续可拆分：

- 基于 GraphRAG 的课程知识点证据召回与推荐解释方法。
- 基于工具调用智能体的个性化学习路径规划方法。
- 面向前沿论文增量融合的课程知识图谱更新方法。

## 七、最终建议

短期最优实现：

> 不要直接做完整 DQN，也不要让 LLM 裸推荐。先做“KT 状态 + KG 约束 + 可解释打分器 + 标准评测框架”。

这条路线与当前项目耦合度最低、论文故事最稳、专利保护点最清楚，也最容易在后续扩展为 Bandit/RL 和 GraphRAG Agent。
