# A3 多智能体个性化资源生成 —— 架构与提交材料

> 配套文档：`a3-implemented-features-summary.md`（功能清单）、`ai-tools-and-licenses.md`（开源与 AI 工具说明）、`china-software-cup-a3-problem.md`（赛题原文）。
> 本文用于初赛提交材料：多智能体协同框架图、资源生成流程图、学生画像维度说明、五类资源样例与防幻觉/内容安全说明。

## 一、系统总体架构

```mermaid
flowchart LR
  subgraph FE[前端 Vue3]
    A3[A3 学习智能体工作台]
    RC[资源中心]
    KD[知识详情/图谱]
  end
  subgraph BE[后端 Go + Gin]
    PROF[profile 画像模块]
    RG[resourcegen 资源生成模块]
    REC[recommend 推荐模块]
    PLAN[studyplan 学习计划]
    GRAPH[graph 知识图谱]
  end
  subgraph INFRA[基础设施]
    MYSQL[(MySQL)]
    NEO4J[(Neo4j)]
    REDIS[(Redis)]
    LLM[DeepSeek / OpenAI 兼容大模型]
  end

  A3 --> PROF
  A3 --> PLAN
  A3 --> RG
  RC --> RG
  KD --> RG
  PLAN --> REC
  REC --> GRAPH
  RG --> PROF
  RG --> REC
  RG --> GRAPH
  PROF --> MYSQL
  RG --> MYSQL
  GRAPH --> NEO4J
  REC --> REDIS
  RG --> LLM
  PROF --> LLM
```

## 二、多智能体协同框架

资源生成由 `resourcegen` 模块编排，多个角色智能体按序协作，过程对前端可见（步骤状态 + 进度 + SSE 流式推送）。

```mermaid
flowchart TD
  ORCH[Orchestrator 编排器] --> VIS[VisionReaderAgent 多模态识别（含图片时）]
  VIS --> PA[ProfileAnalystAgent 画像分析]
  ORCH --> PA
  PA --> CP[CoursePlannerAgent 课程规划]
  CP --> RW[ResourceWriterAgents 资源生成智能体组]
  RW --> CWA[ContentWriterAgent 讲解文档]
  RW --> MMA[MindMapAgent 思维导图]
  RW --> QA[QuestionAgent 练习题]
  RW --> CA[CaseAgent 代码案例]
  RW --> RA[ReadingAgent 拓展阅读]
  RW --> MVA[MultimodalAgent 视频脚本]
  CWA --> SR[SafetyReviewerAgent 安全与防幻觉审查]
  MMA --> SR
  QA --> SR
  CA --> SR
  RA --> SR
  MVA --> SR
  SR --> ARCH[资源落库 + 推送流 + 学习事件]
```

| 智能体 | 职责 | 输入 | 输出 |
| --- | --- | --- | --- |
| Orchestrator | 流程编排、进度推进、失败兜底 | 生成请求 | 任务状态机 |
| VisionReaderAgent | 读取上传图片，生成文字摘要 | 题图/笔记/教材页 | 视觉摘要（注入后续上下文）|
| ProfileAnalystAgent | 读取六维画像与学习目标 | 学生画像 | 生成需求 |
| CoursePlannerAgent | 结合知识图谱与推荐路径定知识点与顺序 | 图谱、推荐路径 | 学习范围与顺序 |
| ResourceWriterAgents | 并发生成多类资源 | 上下文 + 大模型 | 各类资源正文 |
| SafetyReviewerAgent | 事实性/防幻觉/敏感/格式/教学适配审查 | 资源正文 | 审查报告 |

## 三、资源生成流程（异步 + 流式）

```mermaid
sequenceDiagram
  participant U as 学生
  participant FE as A3 工作台
  participant API as resourcegen API
  participant BG as 后台协程(多智能体)
  participant LLM as 大模型
  U->>FE: 选择知识点/上传图片/点击生成
  FE->>API: POST /resources/generate
  API->>BG: 启动异步任务(detached ctx)
  API-->>FE: 立即返回 running 任务
  FE->>API: GET /resources/jobs/:id/stream (SSE)
  loop 每 ~700ms
    BG->>LLM: 各智能体生成/审查
    BG-->>API: 更新进度与步骤(落库)
    API-->>FE: data: {job 快照}
  end
  BG-->>API: 任务 completed
  API-->>FE: data: [DONE]
  FE-->>U: 富渲染资源 + 反馈入口 + 效果评估
```

## 四、对话式学生画像（六维）

学生用一段自然语言描述学习情况，系统抽取 6 个结构化维度，每维含维度值、置信度、证据，并随学习行为更新（随学随新）。

| 维度 key | 名称 | 含义 |
| --- | --- | --- |
| knowledge_foundation | 知识基础 | 已掌握内容与薄弱点 |
| learning_goal | 学习目标 | 目标方向与诉求 |
| learning_progress | 学习进度 | 当前所处阶段与近期完成 |
| cognitive_style | 认知风格 | 项目驱动 / 视觉结构 / 理论优先等 |
| error_preference | 易错点偏好 | 常见错误与需强化的提醒形式 |
| resource_preference | 资源偏好 | 偏好的资源类型 |

> 更新触发：资源生成、资源浏览、资源反馈、练习答题等行为写入 `learning_events`，`POST /api/profile/refresh` 据此刷新维度。

## 五、五类资源样例

1. **课程讲解文档（Markdown）**：知识点定义、原理、要点、示例、小结。
2. **知识点思维导图（Mermaid mindmap）**：中心主题 + 分支知识结构，前端可渲染。
3. **练习题目**：单选/简答，含答案与易错解析；并提供结构化互动练习（`POST /resources/quiz`）支持在线答题与效果回写。
4. **代码实操案例**：实验目标、步骤、可运行代码片段、运行说明与思考题。
5. **拓展阅读材料**：延伸主题、阅读理由与学习路径建议（来源标注，避免编造链接）。

> 加分项：多模态短视频/动画讲解脚本（分镜 + 旁白 + 素材提示）。

## 六、防幻觉与内容安全

- **生成约束**：每个资源写作智能体的系统提示词都附加防幻觉规则——只围绕给定知识点与上下文生成、不编造事实/数据/人名/论文/链接、不确定内容标注“需进一步核实”。
- **可解释依据**：资源详情展示依据来源（知识点、课程、生成智能体、个性化依据、任务编号、安全机制）。
- **安全审查智能体**：`SafetyReviewerAgent` 对资源做事实性/防幻觉、敏感与违规过滤、格式完整性、教学适配性检查，输出状态、分数、风险与建议；评分过低自动判为阻断。
- **离线兜底**：大模型不可用时回退本地模板，保证演示可用且不产生不可控内容。
- **隐私边界**：图片仅作为本次生成的多模态输入，不长期留存原图于第三方。

## 七、学习效果评估闭环

```mermaid
flowchart LR
  GEN[生成个性化资源] --> LEARN[学习/答题]
  LEARN --> FB[提交反馈 + 答题事件]
  FB --> EVT[(learning_events)]
  EVT --> REFRESH[画像随学随新]
  EVT --> EFFECT[学习效果评估看板]
  REFRESH --> RECO[反馈感知的下一步推荐]
  EFFECT --> RECO
  RECO --> GEN
```

- 效果评估端点：`GET /api/resources/effect-summary`（生成数、覆盖知识点、完成率、平均评分、有帮助率、难度分布、近期行为与文字洞察）。
- 互动练习：`POST /api/resources/quiz` 出结构化单选题，答题结果以 `practice_answered` 事件回写，驱动易错点偏好与效果评估更新。
