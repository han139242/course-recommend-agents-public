# A3 赛题功能达标差距与实施计划

> 对照赛题原文：`docs/competition/china-software-cup-a3-problem.md`
> 阅读范围：`backend/internal/bootstrap`、`backend/internal/router`、`backend/internal/module/{assistant,studyplan,recommend,learning,competency,dashboard,frontier,evaluation,graph,teacher}`、`backend/migrations`、`frontend/src/{router.js,api,stores,views,components,composables}`、现有比赛文档。

## 一句话结论

当前系统已经具备“知识图谱 + 学习进度 + 能力画像 + 路径推荐 + AI 助教 + 练习反馈”的学习闭环雏形，但还没有达到 A3 赛题要求的“基于大模型的个性化资源生成智能体系统”。要达标，核心工作不是再美化图谱，而是补上三条主链路：

1. 对话式学生画像：从自然语言持续抽取不少于 6 个维度，并随学习行为更新。
2. 多智能体资源生成：由不同角色智能体协作生成并保存至少 5 类个性化学习资源。
3. 资源推送与效果闭环：把生成资源绑定到学习路径、推送给学生，并根据使用和测评反馈持续调整。

## 当前已有能力

### 已有后端能力

- `assistant` 模块：有 `/api/assistant/chat`、`/api/assistant/public-chat` 和 `/api/assistant/knowledge-card`。服务内会按用户意图分成“学习规划助手、知识探索助手、评估诊断助手”，并可调用图谱、学习进度、能力画像、推荐路径、前沿摘要等工具。
- `studyplan` 模块：有目标画像表单、学习目标保存、三阶段学习计划生成，接口为 `/api/study-plan/profile`、`/api/study-plan/current`、`/api/study-plan/generate`。
- `recommend` 模块：有 KT-KGPath 评分器，按薄弱度、先修满足度、目标相关度、遗忘风险、跨课程迁移、前沿关联度等因子生成推荐路径。
- `learning` 模块：支持教师导入单选题、学生答题、测验提交、Redis/MySQL 掌握度快照和学习进度查询。
- `competency` 与 `dashboard` 模块：能把掌握度映射为能力雷达、薄弱能力、成长洞察，并在仪表盘聚合推荐路径。
- `graph` 与 `frontier` 模块：能查询 Neo4j 知识图谱、知识详情、先修链、前沿资料摘要。
- `evaluation` 模块：已有推荐算法评测 baseline 和指标聚合。

### 已有前端能力

- `GraphHome`：知识图谱主界面，包含搜索、课程筛选、节点展开、AI 助教、学习目标向导、学习计划卡片。
- `KnowledgeDetail`：知识点详情、前置知识、典型场景、推荐路径、练习题、AI 追问、前沿资料。
- `DashboardView`：学习总览、能力画像、薄弱知识点、推荐任务。
- `AiAssistant`：支持流式问答、推荐路径提示、学习提醒、图谱 UI 动作。
- `StudyGoalWizard`：表单式学习目标采集，字段包括年级、专业、目标类型、目标方向、已学内容、基础水平、每周学习时间、学习偏好。

## 逐项差距

| 赛题要求 | 当前状态 | 缺口判断 | 需要补的能力 |
| --- | --- | --- | --- |
| 对话式学习画像自主构建，不少于 6 个维度，随学随新 | 有表单式目标画像；`assistant` 可从一句话推断目标草稿；`competency` 可由掌握度生成能力画像 | 部分满足，但不达标 | 需要真正的对话式画像抽取、画像维度标准化、证据来源、置信度、版本更新和学习行为触发刷新 |
| 多智能体协同的资源生成，至少 5 种资源 | 只有单助手工具调用和知识卡片生成；没有资源生成任务、资源库、协作流程 | 主要缺失 | 需要新增资源生成模块、智能体角色、任务编排、异步进度、资源落库、资源预览与下载 |
| 个性化学习路径规划和资源推送 | 有推荐路径和三阶段学习计划 | 部分满足 | 需要把生成资源绑定到路径节点，并形成推送队列、资源卡片、学习任务状态 |
| 智能辅导（可选加分） | 有 AI 助教、图谱工具、学习数据工具 | 基础可用 | 需要多模态答疑形态，如图解、短视频脚本/讲解卡、生成引用依据和安全过滤 |
| 学习效果评估（可选加分） | 有答题记录、掌握度、能力画像、推荐评测 | 部分满足 | 需要记录资源使用行为、资源反馈、学习前后对比，并据此动态调整资源推送 |
| 非功能：流式输出、Markdown、多模态内容卡片化、生成进度 | 已有 AI 流式问答和 Markdown 倾向 | 部分满足 | 资源生成需要独立的进度追踪、长任务状态、失败重试、卡片化资源展示 |
| 防幻觉与内容安全 | 助手提示要求不编造；前沿资料做可信来源过滤 | 不足 | 需要资源级引用、知识库检索约束、事实校验、安全词过滤、人工可审阅记录 |
| 开源/AI 工具说明，科大讯飞相关工具要求 | 暂未系统化整理 | 缺失 | 需要新增 `docs/competition/ai-tools-and-licenses.md`，列明 AI Coding、模型、框架、开源协议和科大讯飞工具使用说明 |

## 必须新增的后端能力

### 1. 学生画像模块升级

建议新增或扩展为 `backend/internal/module/profile`，不要把所有逻辑继续塞进 `studyplan`。

需要的数据模型：

- `student_learning_profiles`：学生画像主表。
- `student_profile_dimensions`：画像维度明细，至少包含知识基础、认知风格、易错点偏好、学习目标、学习节奏、资源偏好；可以扩展为专业方向、学习历史、工具熟悉度、项目偏好。
- `student_profile_evidence`：画像依据，记录来源是对话、答题、练习、资源反馈还是教师导入。
- `student_profile_versions`：画像版本，用于展示“随学随新”。

建议接口：

- `POST /api/profile/extract`：输入自然语言对话，抽取画像草稿。
- `GET /api/profile/current`：读取当前完整画像。
- `PUT /api/profile/current`：学生确认或编辑画像。
- `POST /api/profile/refresh`：根据最新学习行为刷新画像。
- `GET /api/profile/timeline`：展示画像变化历史。

实现要点：

- 复用 `assistant/service/action_intent.go` 中已有的目标画像推断逻辑，但要升级成后端正式接口。
- 画像维度必须结构化，不能只保存一段目标文本。
- 每个维度保存 `value`、`confidence`、`evidence`、`updatedAt`。
- 答题、资源点击、收藏、笔记、学习计划完成都应触发画像更新事件。

### 2. 多智能体资源生成模块

建议新增 `backend/internal/module/resourcegen`，这是达标的最大缺口。

需要的数据模型：

- `resource_generation_jobs`：生成任务，记录学生、课程、知识点、目标、状态、进度、错误信息。
- `generated_resources`：生成资源，记录类型、标题、内容、格式、关联知识点、来源、版本。
- `resource_agent_steps`：多智能体步骤日志，记录每个角色的输入、输出摘要、耗时、状态。
- `resource_feedback`：学生对资源的评分、是否有帮助、是否太难、是否继续学习。
- `resource_push_items`：资源推送队列，绑定学习路径、推荐理由、展示顺序。

建议接口：

- `POST /api/resources/generate`：创建资源生成任务。
- `GET /api/resources/jobs/:jobId`：查询进度和智能体步骤。
- `GET /api/resources`：查询资源库，支持按知识点、类型、课程、状态过滤。
- `GET /api/resources/:resourceId`：查看单个资源。
- `POST /api/resources/:resourceId/feedback`：提交资源反馈。
- `POST /api/resources/:resourceId/regenerate`：基于反馈重新生成。
- `GET /api/resources/feed`：读取当前学生的个性化资源推送流。

建议智能体角色：

- `ProfileAnalystAgent`：读取学生画像、掌握度、目标，形成资源生成需求。
- `CoursePlannerAgent`：从 Neo4j 知识图谱选择知识点范围、先修链和学习顺序。
- `ContentWriterAgent`：生成专业课程讲解文档。
- `QuestionAgent`：生成不同类型练习题、答案和解析。
- `MindMapAgent`：生成知识点思维导图，建议输出 Mermaid mindmap 或结构化 JSON。
- `ReadingAgent`：生成拓展阅读材料和参考资料。
- `CaseAgent`：生成代码类实操案例、实验步骤、运行说明。
- `MultimodalAgent`：生成教学视频/动画脚本、分镜、旁白、素材提示；初赛阶段可先交付脚本和可视化卡片。
- `SafetyReviewerAgent`：做事实校验、引用检查、敏感内容过滤和格式检查。
- `Orchestrator`：控制流程、进度、失败重试和最终资源归档。

至少 5 类资源建议先做这些：

1. 专业课程讲解文档：Markdown。
2. 知识点思维导图：Mermaid mindmap 或 JSON，前端可渲染。
3. 练习题目：单选/判断/简答至少两类，初赛可先单选 + 简答。
4. 拓展阅读材料：带来源说明和学习理由。
5. 代码类实操案例：实验目标、步骤、代码、运行说明、思考题。
6. 多模态教学视频/动画脚本：作为加分项，先生成分镜脚本和旁白。

### 3. 个性化资源推送

当前 `recommend` 只推“知识点路径”，还没有推“资源”。需要把路径和资源打通：

- 推荐路径中的每个 `PathItemView` 追加可用资源列表。
- 生成资源后写入 `resource_push_items`，按学生画像、薄弱点、学习偏好排序。
- 仪表盘展示“今日资源包”“正在生成”“建议复习资源”。
- 知识详情页展示该知识点下的讲解文档、导图、题目、案例、视频脚本。
- 学习计划卡片从“知识点三阶段”升级为“阶段 + 资源包 + 完成状态”。

### 4. 学习行为与效果评估

当前掌握度主要来自答题。A3 要求跟踪学习行为、练习测试、资源使用反馈，需要补事件表和分析逻辑。

建议新增：

- `learning_events`：资源浏览、停留、收藏、笔记、下载、完成、跳过、重新生成。
- `resource_learning_outcomes`：资源学习前后的掌握度变化、练习正确率变化、反馈评分。
- `plan_adjustment_logs`：因评估结果调整资源或计划的记录。

建议接口：

- `POST /api/learning/events`：前端统一上报资源使用事件。
- `GET /api/learning/effect-summary`：学习效果汇总。
- `POST /api/study-plan/adjust`：根据最新画像和效果重新调整计划。

## 必须新增的前端能力

### 1. 对话式画像页或抽屉

当前 `StudyGoalWizard` 是表单，不能充分体现“摒弃传统繁琐表单”。建议做成“对话 + 画像确认”双栏：

- 左侧是自然语言对话：学生描述专业、目标、历史、困难。
- 右侧实时生成画像维度卡片：知识基础、认知风格、易错点偏好、学习目标、学习节奏、资源偏好等。
- 每张卡片显示证据、置信度、可编辑字段。
- 点击确认后写入正式画像，再生成资源和路径。

可复用：

- `AiAssistant` 的流式展示。
- `StudyGoalWizard` 的字段和保存流程。
- `assistant` 已有的 `propose_goal_profile` 动作意图。

### 2. 资源生成中心

建议新增路由：

- `/resources`：资源库与生成中心。
- `/resources/:resourceId`：资源详情。
- `/generate` 或首页右侧抽屉：发起多智能体资源生成。

界面应包含：

- 资源类型选择：讲解文档、思维导图、练习题、拓展阅读、实操案例、视频脚本。
- 生成依据：当前画像、目标课程、知识点、薄弱点、学习偏好。
- 多智能体流程条：画像分析、课程规划、内容生成、题目生成、校验归档。
- 生成进度：百分比、当前智能体、预计剩余、失败重试。
- 资源卡片：类型、适合原因、关联知识点、预计学习时长、完成状态。

### 3. 知识详情页增强

`KnowledgeDetail` 现在有练习和 AI 追问，建议新增“个性化资源包”区域：

- 当前知识点的讲解文档。
- 思维导图预览。
- 针对画像生成的练习题。
- 实操案例入口。
- 视频/动画脚本预览。
- 对每个资源提供“有帮助/太难/太浅/重新生成”反馈。

### 4. 仪表盘增强

`DashboardView` 应增加：

- 今日推荐资源。
- 资源完成度。
- 资源使用反馈对学习计划的影响。
- 最近生成任务状态。
- 学习效果评估：资源学习前后正确率变化、掌握度变化。

## 非功能和提交材料需要补齐

### 防幻觉与内容安全

需要在资源生成链路中加入明确机制：

- 每个资源必须记录引用的知识点、图谱节点、题库或资料来源。
- 生成后由 `SafetyReviewerAgent` 检查事实性、敏感信息和格式。
- 前端展示“依据来源”，避免看起来像纯模型编造。
- 对学术内容做知识图谱术语校验：关键概念必须能在知识库中找到或标记为外部概念。

### 响应时间与长任务

多模态资源生成不能同步阻塞 HTTP 请求。需要：

- 后端异步任务和状态查询。
- 前端生成进度追踪。
- SSE 或轮询推送状态。
- 失败重试和部分产物保留。

### 开源与 AI 工具说明

需要新增文档：

- `docs/competition/ai-tools-and-licenses.md`
- 说明 Go、Gin、Vue、Vite、Pinia、Neo4j、Redis、GORM、lucide、KaTeX、AntV/G6 或现有图谱渲染依赖。
- 说明使用的 AI 模型、AI Coding 工具、科大讯飞相关工具。
- 标注许可证、来源链接、用途和是否参与最终运行。

### 初赛提交材料

需要补齐：

- 多智能体协同框架图。
- 资源生成流程图。
- 学生画像维度说明。
- 5 类资源样例。
- 运行说明和数据集说明。
- 防幻觉与内容安全说明。
- AI Coding 工具使用说明。

## 推荐实施路线

### P0：初赛达标最小闭环

目标：让评委能看到 A3 要求的核心功能都可操作。

1. 新增 `resourcegen` 后端模块和三张核心表：`resource_generation_jobs`、`generated_resources`、`resource_agent_steps`。
2. 先实现 5 类文本/结构化资源：讲解文档、思维导图、练习题、拓展阅读、实操案例。
3. 新增资源生成中心页面，能发起任务、看进度、看结果。
4. 把资源结果挂到知识详情页和仪表盘。
5. 把 `StudyGoalWizard` 升级为“对话抽取画像 + 确认画像”，至少展示 6 个维度。
6. 文档补齐多智能体流程、AI 工具说明、防幻觉说明。

### P1：增强可解释与个性化

1. 资源生成前强制读取学生画像、掌握度、推荐路径和知识图谱。
2. 每个资源展示“为什么推荐给你”。
3. 资源反馈写入 `resource_feedback`，影响后续推送。
4. 生成任务支持 SSE 进度。
5. 练习题生成后可进入现有答题和掌握度更新链路。

### P2：冲击加分项

1. 智能辅导支持图解说明、短视频脚本、动画分镜。
2. 学习效果评估加入资源使用前后对比。
3. 自动调整学习计划并展示调整原因。
4. 引入科大讯飞相关工具作为可说明的 AI 辅助工具链。
5. 生成 PPT 大纲或演示素材，直接服务比赛提交。

## 建议新增目录和文件

```text
backend/internal/module/profile/
backend/internal/module/resourcegen/
backend/migrations/000008_create_student_learning_profiles.sql
backend/migrations/000009_create_generated_resources.sql
backend/migrations/000010_create_learning_events.sql
frontend/src/api/resources.js
frontend/src/api/profile.js
frontend/src/views/ResourceCenterView.vue
frontend/src/views/ResourceDetailView.vue
frontend/src/components/profile/ConversationalProfileBuilder.vue
frontend/src/components/resources/ResourceGenerationPanel.vue
frontend/src/components/resources/ResourceCard.vue
frontend/src/components/resources/AgentProgressTimeline.vue
docs/competition/ai-tools-and-licenses.md
docs/competition/a3-demo-script.md
```

## 验收标准

### 功能验收

- 学生能用自然语言描述目标，系统生成不少于 6 个画像维度，并可确认保存。
- 学生选择课程/知识点后，系统能发起一次多智能体资源生成任务。
- 任务过程能显示至少 4 个智能体步骤和进度。
- 任务完成后至少生成 5 类资源，并能在资源库和知识详情页查看。
- 推荐路径能展示绑定资源，不只是知识点列表。
- 学生对资源做反馈后，仪表盘或路径建议能体现调整。

### 演示验收

推荐 7 分钟演示顺序：

1. 展示对话式画像构建：输入“我是大二人工智能学生，想做智能体项目，机器学习基础不稳”。
2. 展示画像卡片：知识基础、认知风格、易错点偏好、目标方向、学习节奏、资源偏好。
3. 点击“生成个性化资源包”，展示多智能体步骤。
4. 展示 5 类资源：文档、导图、练习题、拓展阅读、实操案例。
5. 打开推荐路径，说明资源如何绑定到学习顺序。
6. 做一道练习题，展示掌握度和能力画像更新。
7. 展示防幻觉说明、来源依据和 AI 工具/开源协议说明。

## 风险提示

- 如果只保留当前“知识图谱推荐系统”定位，会偏离 A3 赛题的“资源生成智能体系统”核心。
- 如果多智能体只停留在“助手显示三个角色名称”，评委可能认为没有真实协同框架。
- 如果生成结果不落库、不展示进度、不绑定学生画像，就很难证明“个性化资源体系”。
- 如果没有开源协议和 AI Coding 工具说明，会踩提交材料要求。
- 如果没有防幻觉与内容安全链路，生成学术内容的可信度会被质疑。
