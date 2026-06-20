# AI 工具与开源协议说明

本文用于中国软件杯 A3 赛题提交材料，说明系统中使用的 AI 工具、开源组件、许可证与内容安全约束。

## 1. AI 工具使用说明

| 工具/模型 | 用途 | 是否核心运行依赖 | 说明 |
| --- | --- | --- | --- |
| DeepSeek / OpenAI 兼容接口 | 对话式画像抽取、资源生成、资源安全审查、下一步推荐 | 可选 | 通过后端 `.env` 中的 `OPENAI_*` 或 `DEEPSEEK_*` 变量配置。未配置时系统使用本地规则和模板兜底，保证离线演示可运行。 |
| Vision-capable LLM 接口 | 读取教材页、题目截图、笔记图片，形成 `VisionReaderAgent` 摘要 | 可选 | 仅在模型支持视觉输入且学生上传图片时启用。 |
| Codex / AI Coding 工具 | 辅助代码实现、重构和文档整理 | 否 | 产出代码由团队人工审阅、测试和提交。 |
| 科大讯飞相关工具 | 参赛 PPT 文案润色、演示视频字幕校对、答辩材料语言优化 | 否 | 不作为系统线上核心服务依赖；若后续接入讯飞星火等模型接口，应在 `.env` 和接口文档中补充配置方式。 |

## 2. 多智能体框架说明

系统在 `backend/internal/module/resourcegen` 中实现资源生成编排，关键智能体包括：

| 智能体 | 职责 |
| --- | --- |
| `VisionReaderAgent` | 读取学生上传图片，提取题目、笔记或教材页要点。 |
| `ProfileAnalystAgent` | 读取学生画像、目标、偏好和行为信号，形成个性化生成需求。 |
| `CoursePlannerAgent` | 结合知识图谱、先修链和推荐路径，确定资源关联知识点。 |
| `ResourceWriterAgents` | 协同生成讲解文档、思维导图、练习题、拓展阅读、代码案例、视频脚本。 |
| `SafetyReviewerAgent` | 检查事实性、防幻觉、敏感内容、格式完整性和教学适配性。 |

## 3. 主要开源组件与许可证

| 类别 | 组件 | 版本来源 | 常见许可证 | 项目用途 |
| --- | --- | --- | --- | --- |
| 后端 Web 框架 | Gin | `backend/go.mod` | MIT | HTTP API、路由和中间件。 |
| ORM | GORM / MySQL Driver | `backend/go.mod` | MIT | MySQL 数据访问。 |
| 图数据库驱动 | Neo4j Go Driver | `backend/go.mod` | Apache-2.0 | 知识图谱查询。 |
| 缓存客户端 | go-redis | `backend/go.mod` | BSD-2-Clause | 学习状态缓存与运行辅助。 |
| 日志 | zap | `backend/go.mod` | MIT | 结构化日志。 |
| 前端框架 | Vue 3 | `frontend/package.json` | MIT | 学生端与展示端 UI。 |
| 构建工具 | Vite | `frontend/package.json` | MIT | 前端开发与生产构建。 |
| 状态管理 | Pinia | `frontend/package.json` | MIT | 用户、图谱和学习状态管理。 |
| 路由 | Vue Router | `frontend/package.json` | MIT | 页面路由。 |
| 图谱交互 | D3 系列包 | `frontend/package.json` | ISC | 图谱布局、拖拽、缩放和空间索引。 |
| 图标 | Lucide Vue | `frontend/package.json` | ISC | 界面图标。 |
| 数学公式 | KaTeX | `frontend/package.json` | MIT | 资源和 AI 回复中的公式渲染。 |
| 测试 | Vitest / Vue Test Utils / jsdom | `frontend/package.json` | MIT | 前端单元测试与组件测试。 |

实际提交前应以 `go.mod`、`go.sum`、`package.json`、`package-lock.json` 中的版本为准，并保留原组件许可证声明。

## 4. 内容安全与防幻觉机制

- 资源生成提示词要求只围绕知识点、画像、图谱详情和推荐路径生成内容。
- 阅读材料不杜撰具体论文、书名或链接；不确定内容标注“需进一步核实”。
- 资源详情页展示依据来源，包括知识点、课程、生成智能体、个性化原因、任务编号和安全机制。
- `SafetyReviewerAgent` 对每份资源生成事实性、防幻觉、敏感过滤、格式完整性和教学适配性报告。
- 导出的 Markdown 资源包自动附带安全审查摘要，便于人工复核和提交归档。

## 5. 数据与隐私边界

- AI 密钥仅放在后端 `.env`，前端不保存模型密钥。
- 学生画像、资源反馈和学习事件只在登录用户范围内查询。
- 学习事件用于画像刷新和资源推送排序，不包含敏感身份信息之外的额外隐私字段。
