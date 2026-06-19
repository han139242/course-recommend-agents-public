# A3 赛题功能实现说明

更新时间：2026-06-19

本文档记录本轮围绕“中国软件杯 A3 基于大模型的个性化资源生成”赛题已完成的两个核心功能：对话式学生画像构建、多智能体个性化资源生成。

## 功能 1：对话式学生画像构建

已实现一个独立的 `profile` 后端模块，用于把学生自然语言描述转化为结构化学习画像，并持久化为当前画像。

### 已完成能力

- 支持学生输入自然语言学习情况，例如专业基础、学习目标、已学内容、薄弱点、资源偏好。
- 自动抽取 6 个画像维度：
  - 知识基础
  - 学习目标
  - 学习进度
  - 认知风格
  - 易错点偏好
  - 资源偏好
- 每个维度包含维度值、置信度、证据文本，便于前端展示和后续资源生成引用。
- 支持保存确认后的画像，并保留画像版本记录。
- 支持读取当前学生画像，作为资源生成的个性化上下文。

### 后端接口

- `POST /api/profile/extract`：根据自然语言描述生成画像。
- `GET /api/profile/current`：读取当前学生画像。
- `PUT /api/profile/current`：保存确认后的画像。
- `POST /api/profile/refresh`：刷新当前画像读取结果。

### 数据表

新增迁移文件：`backend/migrations/000008_create_learning_profiles.sql`

- `student_learning_profiles`：存储学生当前画像摘要和原始输入。
- `student_profile_dimensions`：存储 6 个画像维度及证据。
- `student_profile_versions`：记录画像变更历史。

### 前端入口

新增页面：`frontend/src/views/ResourceCenterView.vue`

页面左侧为“对话式学生画像”区域，支持输入学习情况、生成画像、展示 6 维画像卡片、确认保存画像。

## 功能 2：多智能体个性化资源生成

已实现一个独立的 `resourcegen` 后端模块，用于基于学生画像、知识点信息和推荐路径生成多类型学习资源。

### 已完成能力

- 支持选择知识点名称、知识点 ID、课程名称、生成目标和资源类型。
- 生成过程以多智能体步骤显式展示：
  - `ProfileAnalystAgent`：读取学生画像、学习目标和资源偏好。
  - `CoursePlannerAgent`：结合知识图谱和推荐路径，确定知识点及学习顺序。
  - `ResourceWriterAgents`：协同生成多种学习资源。
  - `SafetyReviewerAgent`：检查资源格式、学习依据和内容安全。
- 支持生成以下资源类型：
  - 课程讲解文档
  - 知识点思维导图
  - 个性化练习题
  - 拓展阅读材料
  - 代码类实操案例
  - 多模态短视频讲解脚本（可选）
- 生成结果会进入学生资源库，页面可刷新查看历史资源。

### 后端接口

- `POST /api/resources/generate`：生成个性化资源包。
- `GET /api/resources/jobs/:jobId`：查询资源生成任务详情。
- `GET /api/resources`：查询当前学生资源库。
- `GET /api/resources/:resourceId`：查询单个资源详情。

### 数据表

新增迁移文件：`backend/migrations/000009_create_generated_resources.sql`

- `resource_generation_jobs`：记录一次资源生成任务。
- `resource_agent_steps`：记录多智能体协作步骤。
- `generated_resources`：存储生成后的资源内容。

### 前端入口

新增页面：`frontend/src/views/ResourceCenterView.vue`

页面右侧为“多智能体资源生成”区域，支持填写生成参数、勾选资源类型、查看智能体步骤进度、查看本次生成结果。页面侧栏提供资源库列表。

首页 `GraphHome.vue` 已新增“资源生成”入口，登录后可直接进入 `/resources`。

## 系统接入情况

- 后端模块已接入 `bootstrap.New()` 组合根。
- 后端路由已挂载到登录鉴权后的 `/api/profile` 和 `/api/resources`。
- 前端已新增 `frontend/src/api/profile.js` 和 `frontend/src/api/resources.js`。
- 前端路由已新增 `/resources`，并启用登录鉴权。
- 已重新执行前端生产构建，更新 `backend/web/dist` 静态产物。

## 验证结果

已完成以下验证：

- `go test ./...`：通过。
- `npm ci`：通过。
- `npm run build`：通过。

说明：`npm ci` 输出了现有依赖的安全审计提示，本次未执行 `npm audit fix`，避免引入额外依赖版本变更。

## 当前实现边界

本轮实现的是可演示、可持久化的 MVP 闭环。当前画像抽取和资源生成采用规则推理与模板生成方式，已经具备比赛演示中的功能路径。后续若继续增强，可接入 DeepSeek/大模型真实生成、多轮追问补全画像、异步任务队列、流式生成进度、资源质量评分与学生反馈闭环。
