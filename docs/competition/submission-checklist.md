# 初赛提交材料清单与演示物料指引

> 中国软件杯 A3 赛题提交自查表。配套：`a3-architecture.md`、`a3-implemented-features-summary.md`、`deployment-and-dataset.md`、`ai-tools-and-licenses.md`、`china-software-cup-a3-problem.md`、`selection-video-script.md`。

## 一、提交材料清单（自查表）

| 项 | 内容 | 状态 | 位置 / 产出方式 |
| --- | --- | --- | --- |
| 系统说明文档 | 功能、架构、亮点 | ✅ | `a3-implemented-features-summary.md`、`a3-architecture.md` |
| 多智能体协同框架图 | 智能体角色与协作 | ✅ | `a3-architecture.md` 第二节（Mermaid）|
| 资源生成流程图 | 异步 + SSE 时序 | ✅ | `a3-architecture.md` 第三节（Mermaid）|
| 学生画像维度说明 | 六维定义与更新触发 | ✅ | `a3-architecture.md` 第四节 |
| 五类资源样例 | 文档/导图/练习/案例/阅读 | ✅ | `a3-architecture.md` 第五节；系统内实时生成 |
| 防幻觉与内容安全说明 | 约束/审查/依据 | ✅ | `a3-architecture.md` 第六节；`ai-tools-and-licenses.md` 第四节 |
| 运行说明 | 部署、启动、测试 | ✅ | `deployment-and-dataset.md` |
| 数据集说明 | 图谱/关系库/缓存来源 | ✅ | `deployment-and-dataset.md` 第六节 |
| AI 工具与开源协议说明 | 模型、AI Coding、讯飞、许可证 | ✅ | `ai-tools-and-licenses.md` |
| 演示视频 | 7 分钟系统演示 | ⏳ 待录制 | 脚本：`selection-video-script.md`；系统内可生成演示脚本 |
| 答辩 PPT | 演示提纲 | ⏳ 待制作 | 系统内「生成 PPT 大纲」可一键产出提纲骨架 |

> ✅ = 已就绪（文档/能力已具备）；⏳ = 需团队基于现有产出完成成品（录制/制作）。

## 二、系统内置的物料生成能力

资源生成任务完成后，A3 工作台与资源中心提供一键产出，可直接作为提交/答辩素材底稿：

| 能力 | 入口 | 接口 | 用途 |
| --- | --- | --- | --- |
| PPT 大纲 | A3 任务区「生成 PPT 大纲」 | `GET /api/resources/jobs/:id/presentation-outline` | 答辩 PPT 骨架 |
| 演示脚本 | A3 任务区「生成演示脚本」 | `GET /api/resources/jobs/:id/demo-script` | 7 分钟演示视频分镜与讲稿 |
| Markdown 资源包 | A3 任务区「导出 Markdown」 | `GET /api/resources/jobs/:id/export` | 资源样例附件，含安全审查摘要 |

## 三、五类资源样例获取方式

在 `/a3-agent` 选定知识点（如「智能体与工具调用」），勾选全部资源类型并生成，即可得到：

1. 课程讲解文档（Markdown）
2. 知识点思维导图（Mermaid，前端渲染）
3. 练习题目（含解析）+ 互动随堂练习（在线作答、回写掌握度）
4. 代码实操案例
5. 拓展阅读材料

> 导出 Markdown 资源包即可作为「五类资源样例」附件提交，每份附安全审查摘要。

## 四、建议演示顺序（约 7 分钟）

详见 `selection-video-script.md`。要点对应系统操作：

1. 自然语言输入 → 六维画像（对话式画像，摒弃繁琐表单）。
2. 生成三阶段学习计划 + 下一步推荐（带推荐依据评分）。
3. 选知识点 / 上传图片 → 多智能体资源生成（SSE 实时进度、智能体步骤）。
4. 展示五类资源 + 依据来源 + 安全审查（防幻觉可信）。
5. 随堂练习作答 → 掌握度/画像更新 → 提交反馈 → 学习效果评估 → 按最新掌握度重新规划（自适应闭环）。
6. 一键导出 PPT 大纲 / 演示脚本 / Markdown 资源包。
7. 说明开源协议、AI 工具与防幻觉机制。

## 五、提交前最终检查

- [ ] `go test ./...` 与 `npm run build` 均通过，`backend/web/dist` 为最新构建产物。
- [ ] `.env.example` 覆盖全部必填变量，敏感密钥未入库。
- [ ] Docker 全栈可一键启动并通过 `/healthz`、`/readyz`。
- [ ] 演示视频已按脚本录制，答辩 PPT 已基于大纲完成。
- [ ] 各文档中的接口路径、资源样例与实际系统一致。
