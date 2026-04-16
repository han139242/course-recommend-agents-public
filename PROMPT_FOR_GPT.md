# 知识图谱沉浸式学习平台 — 前端重构完整提示词

> 本文档是一份完整的前端实现指令。请严格按照以下要求，从零构建一个全新的 Vue 3 前端项目。

---

## 一、项目背景

项目名称：**突破课程边界的人工智能专业知识点推荐智能体**

这是一个面向高校学生的知识图谱学习平台。后端已使用 Go (Gin) + Neo4j + MySQL + Redis 完成。你需要构建一个**全新的前端**，用户打开网站第一眼就能看到可交互的知识图谱，通过点击、展开、搜索等方式进行学习。

---

## 二、技术栈要求

| 模块 | 技术 | 版本要求 |
|---|---|---|
| 框架 | Vue 3 (Composition API + `<script setup>`) | ^3.5 |
| 构建 | Vite | ^8 |
| 图谱引擎 | **@antv/G6 v5** (使用 WebGL 渲染器) | ^5 |
| 状态管理 | Pinia | ^3 |
| 路由 | vue-router | ^5 |
| 数学渲染 | KaTeX | ^0.16 |
| HTTP | 原生 fetch（不使用 axios） | — |

**不使用任何 UI 组件库**（不用 Element Plus、Ant Design 等），所有 UI 自行编写 CSS 实现。整体风格为**深色沉浸式**。

---

## 三、后端 API 接口（已实现，直接对接）

所有 API 基础路径为 `/api`，开发时由 Vite 代理到后端（`http://localhost:8080`）。

### 3.1 公开接口（无需 Token）

#### GET `/api/graph/full?limit=2000`

返回完整知识图谱。

```json
{
  "code": 0,
  "data": {
    "nodes": [
      {
        "id": "uuid-xxx",
        "routeKnowledgeId": "梯度下降",
        "name": "梯度下降",
        "canonicalName": "梯度下降",
        "entityType": "algorithm",
        "description": "一种迭代优化算法...",
        "chapterTitle": "机器学习",
        "sourcePath": "machine-learning/optimization",
        "curatedSource": ""
      }
    ],
    "edges": [
      {
        "relationId": "rel-xxx",
        "relationType": "PREREQUISITE_OF",
        "sourceId": "uuid-aaa",
        "targetId": "uuid-bbb",
        "sourceName": "偏导数",
        "targetName": "梯度下降"
      }
    ]
  }
}
```

**节点字段说明：**
- `id`: 唯一标识（UUID）
- `routeKnowledgeId`: 用于路由跳转的友好 ID（通常是中文名）
- `name` / `canonicalName`: 显示名称
- `entityType`: 节点类型，值为 `algorithm` | `concept` | `method` | `application` | `topic`
- `chapterTitle`: 所属课程/章节名称 — **用这个字段做课程分组和颜色区分**
- `sourcePath`: 来源路径
- `description`: 描述

**边字段说明：**
- `relationType`: 关系类型，值为 `PREREQUISITE_OF` | `RELATED_TO` | `SUBTOPIC_OF`
- `sourceId` / `targetId`: 起止节点 ID
- `sourceName` / `targetName`: 起止节点名称

#### GET `/api/graph/subgraph?root=梯度下降&limit=12`

返回以 root 为中心的一跳子图。响应格式同上。

#### GET `/api/knowledge/:knowledgeId`

返回知识点详情。`:knowledgeId` 是 `routeKnowledgeId` 的值（URL 编码的中文名）。

```json
{
  "code": 0,
  "data": {
    "id": "uuid-xxx",
    "routeKnowledgeId": "梯度下降",
    "name": "梯度下降",
    "canonicalName": "梯度下降",
    "entityType": "algorithm",
    "description": "...",
    "chapterTitle": "机器学习",
    "sourcePath": "...",
    "curatedSource": "",
    "summary": "",
    "prerequisites": ["偏导数", "损失函数"],
    "related": ["学习率", "反向传播"],
    "scenarios": ["模型训练", "参数优化"],
    "parentTopics": ["最优化方法"],
    "masteryScore": 65,
    "status": "learning",
    "masteryText": "学习中",
    "recommendReason": "这是深度学习的核心基础"
  }
}
```

#### GET `/api/frontier/briefing?knowledge=梯度下降`

返回前沿研究扩展信息。

### 3.2 需要 Token 的接口

所有请求头加 `Authorization: Bearer <token>`。

#### POST `/api/auth/login`
```json
// Request
{ "username": "student1", "password": "123456" }
// Response data
{ "accessToken": "jwt-xxx", "user": { "id": 1, "username": "student1", "role": "student" } }
```

#### POST `/api/auth/register`
```json
// Request
{ "username": "new_student", "password": "123456", "role": "student" }
// Response 同 login
```

#### GET `/api/auth/me`
返回当前用户信息。

#### GET `/api/recommend/path?target=深度学习&knowledgeId=梯度下降&limit=10&scene=detail`

返回个性化学习路径推荐。

```json
{
  "code": 0,
  "data": {
    "path": [
      {
        "stage": "基础",
        "title": "偏导数",
        "detail": "梯度计算的数学基础",
        "route": "/knowledge/偏导数",
        "knowledgeId": "偏导数",
        "reasonCode": "prerequisite",
        "reasonLabel": "前置知识",
        "masteryScore": 80,
        "correctRate": 0.85,
        "answeredCount": 12,
        "score": 0.92
      }
    ]
  }
}
```

#### GET `/api/learning/progress`
返回学生学习进度列表。

#### GET `/api/dashboard/summary`
返回仪表盘汇总数据。

#### GET `/api/competency/profile`
返回学生能力画像。

#### POST `/api/assistant/chat`
AI 助手流式对话。请求体 `{ messages: [...] }`，返回 SSE 流。

#### POST `/api/assistant/knowledge-card`
AI 知识卡片生成。

### 3.3 需要新增的后端接口（前端先用 localStorage 模拟）

以下接口后端尚未实现，前端先用 **localStorage 持久化**作为临时方案，后续后端补齐后替换：

| 功能 | 模拟存储 Key | 数据结构 |
|---|---|---|
| 收藏节点 | `kg-bookmarks` | `string[]`（routeKnowledgeId 数组） |
| 节点笔记 | `kg-notes` | `Record<string, string>`（knowledgeId → 笔记内容） |
| 自定义学习路径 | `kg-custom-paths` | `{ id: string, name: string, nodes: string[], createdAt: string }[]` |

---

## 四、页面布局结构

整个应用只有一个主页面（全屏沉浸式），布局如下：

```
┌──────────────────────────────────────────────────────────────────────┐
│ [Logo] 🔍 搜索知识点...          [学科筛选▾]   [登录/头像]  [顶部栏] │
├────────────┬─────────────────────────────────┬───────────────────────┤
│            │                                 │                       │
│  左侧面板   │     知识图谱主画布 (G6 Canvas)    │    右侧详情面板        │
│  280px     │     flex: 1                     │    320px              │
│  可折叠     │                                 │    可折叠              │
│            │                                 │                       │
│ ┌────────┐ │  力导向布局，深色背景              │  ┌─────────────────┐  │
│ │知识体系 │ │  网格底纹                         │  │ 节点详情         │  │
│ │学习进度 │ │                                 │  │ (随选中自动更新)  │  │
│ │学习路径 │ │                                 │  │                  │  │
│ │收藏夹   │ │                                 │  └─────────────────┘  │
│ └────────┘ │          [图例] [缩放] [全屏]     │                       │
│            │          [小地图]                 │  🤖 AI助手(悬浮按钮)   │
├────────────┴─────────────────────────────────┴───────────────────────┤
│ 底部: 图例说明                                                        │
└──────────────────────────────────────────────────────────────────────┘
```

整体背景色：`#081018`，暗色系，科技感。

---

## 五、知识图谱主画布（核心模块）

### 5.1 G6 v5 初始化要求

```javascript
// 必须使用 WebGL 渲染器以支持 1000+ 节点流畅渲染
import { Graph, Extensions } from '@antv/g6'

const graph = new Graph({
  container: containerRef.value,
  renderer: 'webgl',  // WebGL 渲染器，关键性能要求
  width: containerWidth,
  height: containerHeight,
  // ... 其余配置见下文
})
```

### 5.2 层级展示机制 — Combo 分组

利用 G6 v5 的 **Combo**（组合节点）功能实现层级展开/折叠：

**第一层（默认展示）** — 课程级 Combo 节点：
- 按节点的 `chapterTitle` 字段分组
- 每个 Combo 显示为大的圆角矩形，内容为：课程名 + 包含知识点数量
- Combo 之间根据内部节点的跨课程关系边自动连线
- 默认**折叠状态**，只显示 Combo 本身

**第二层（点击展开）** — 课程内知识点：
- 点击 Combo → 展开动画 → 内部知识点散开（力导向布局）
- 知识点节点显示为圆形
- 知识点之间显示具体关系连线

**第三层（继续展开）** — 如果某知识点有 `SUBTOPIC_OF` 子节点，可以继续展开

**数据转换逻辑：**

```javascript
// 将后端 flat 数据转换为 G6 Combo 结构
function transformToComboData(nodes, edges) {
  // 1. 按 chapterTitle 分组
  const courseGroups = groupBy(nodes, n => n.chapterTitle || '其他')
  
  // 2. 为每个课程创建 Combo
  const combos = Object.keys(courseGroups).map(course => ({
    id: `combo-${course}`,
    data: {
      label: course,
      nodeCount: courseGroups[course].length,
      // ... 颜色等
    }
  }))
  
  // 3. 每个节点关联到所属 Combo
  const g6Nodes = nodes.map(node => ({
    id: node.id,
    combo: `combo-${node.chapterTitle || '其他'}`,
    data: {
      label: node.canonicalName || node.name,
      entityType: node.entityType,
      chapterTitle: node.chapterTitle,
      routeKnowledgeId: node.routeKnowledgeId,
      // ... 其余字段
    }
  }))
  
  // 4. 转换边
  const g6Edges = edges.map(edge => ({
    id: edge.relationId,
    source: edge.sourceId,
    target: edge.targetId,
    data: {
      relationType: edge.relationType,
      label: RELATION_LABEL_MAP[edge.relationType], // 中文标签
    }
  }))
  
  return { nodes: g6Nodes, edges: g6Edges, combos }
}
```

### 5.3 课程颜色映射

按 `chapterTitle` 自动分配颜色。内置配色表：

```javascript
const COURSE_COLOR_MAP = {
  '机器学习':     { primary: '#4F46E5', light: '#818CF8', dark: '#3730A3' },
  '深度学习':     { primary: '#7C3AED', light: '#A78BFA', dark: '#5B21B6' },
  '数学基础':     { primary: '#0891B2', light: '#22D3EE', dark: '#155E75' },
  '线性代数':     { primary: '#0891B2', light: '#22D3EE', dark: '#155E75' },
  '概率与统计':   { primary: '#0D9488', light: '#2DD4BF', dark: '#115E59' },
  '数据结构':     { primary: '#059669', light: '#34D399', dark: '#064E3B' },
  '自然语言处理': { primary: '#D97706', light: '#FBBF24', dark: '#92400E' },
  '计算机视觉':   { primary: '#DC2626', light: '#F87171', dark: '#991B1B' },
  '大模型':       { primary: '#E11D48', light: '#FB7185', dark: '#9F1239' },
  '知识图谱':     { primary: '#9333EA', light: '#C084FC', dark: '#6B21A8' },
}

// 未匹配的课程自动从备选色板中分配
const FALLBACK_PALETTE = ['#6366F1', '#EC4899', '#F59E0B', '#10B981', '#3B82F6', '#8B5CF6']
```

### 5.4 节点视觉设计

**Combo 节点（课程级，折叠态）：**
- 形状：圆角矩形 (rect)，尺寸约 160×60
- 填充色：课程主色 + 15% 不透明度
- 边框：课程主色，2px
- 文字：课程名（白色粗体 14px） + "(N个知识点)"（灰色 11px）
- 阴影光晕：课程主色，模糊半径 20px，不透明度 0.15

**知识点节点（展开后）：**
- 形状：圆形 (circle)
- 默认半径：根据连接边数映射，范围 16–36px
- 填充色：所属课程主色

**节点学习状态光晕（关键视觉）：**

```
已掌握（mastery >= 80）:
  - 外发光：金色 #FFD700，模糊半径 24px，不透明度 0.6
  - 呼吸动画：光晕不透明度在 0.3 ~ 0.7 之间循环，周期 2.5s
  - 视觉效果类似"点亮的星星"

学习中（40 <= mastery < 80）:
  - 外发光：课程主色，模糊半径 16px，不透明度 0.35
  - 无动画

未学习（mastery < 40 或无数据）:
  - 无光晕
  - 填充色降低不透明度到 0.5
  - 边框为灰色虚线

薄弱点（mastery < 30 且 answeredCount > 0，即学过但没学好）:
  - 外发光：橙色 #F97316，模糊半径 20px
  - 脉冲动画：光晕闪烁，周期 1.5s，提醒关注
```

**实现方式 — G6 v5 自定义节点：**

利用 G6 v5 的 `afterDraw` 钩子在节点后面绘制光晕圆：

```javascript
// G6 v5 自定义节点示例
{
  type: 'knowledge-node',
  afterDraw(model, shapeMap) {
    const { mastery, completed, isWeak } = model.data
    if (completed) {
      // 画金色光晕圆
      const haloShape = new Circle({
        style: {
          r: model.data.size / 2 + 12,
          fill: '#FFD700',
          opacity: 0.5,
          // 配合 G6 动画系统做呼吸效果
        }
      })
      // ... 添加到 shapeMap
    }
  }
}
```

### 5.5 关系线条设计

```javascript
const RELATION_CONFIG = {
  PREREQUISITE_OF: {
    label: '前置知识',
    stroke: '#94A3B8',       // 银灰色
    lineWidth: 2.5,          // 粗线，强关联
    lineDash: [],            // 实线
    endArrow: true,          // 有箭头
    opacity: 0.7,
  },
  SUBTOPIC_OF: {
    label: '属于',
    stroke: '#64748B',       // 暗灰色
    lineWidth: 1.8,
    lineDash: [6, 3],        // 虚线
    endArrow: true,
    opacity: 0.5,
  },
  RELATED_TO: {
    label: '相关',
    stroke: '#475569',       // 更暗
    lineWidth: 1.0,          // 细线，弱关联
    lineDash: [3, 3],        // 点线
    endArrow: false,         // 无箭头（双向）
    opacity: 0.35,
  },
}
```

**关系标签**：默认隐藏，当用户 hover 边或聚焦节点时才显示中文标签，避免视觉杂乱。

### 5.6 交互行为（极其重要）

#### 左键单击节点 — 聚焦模式

```
1. 高亮：该节点 + 所有直接关联的邻居节点 + 连接边
2. 其余所有节点和边：opacity 降至 0.05（几乎不可见）
3. 右侧面板自动更新为该节点的详情
4. 关系标签显示在高亮的边上
5. 再次点击同一节点 → 取消聚焦
6. 点击画布空白处 → 取消聚焦，恢复全部节点
```

#### 左键双击 Combo — 展开/折叠

```
1. 折叠态 Combo 双击 → 展开，内部知识点以力导向动画散开
2. 展开态 Combo 双击 → 折叠回 Combo
3. 展开时，Combo 变为半透明的区域外框（hull），知识点分布其中
4. 未展开的 Combo 之间的连线 = 内部节点跨课程边的汇总
5. 展开后，连线连接到具体的知识点节点
```

#### 右键单击节点 — 上下文菜单

弹出自定义右键菜单，**不使用浏览器默认右键菜单**：

```
┌───────────────────────────┐
│  📌  加入学习路径           │
│  ⭐  收藏 / 取消收藏        │
│  📝  记笔记                │
│  ─────────────────────────│
│  📖  进入详细学习           │  ← 跳转到独立的学习页面
│  🤖  AI 讲解这个知识点      │  ← 打开 AI 助手并预填上下文
│  🔗  查看前置依赖链         │  ← 在图谱上高亮整条前置链
│  ✅  标记为已掌握           │
└───────────────────────────┘
```

实现方式：监听 G6 的 `contextmenu` 事件，阻止浏览器默认行为，渲染一个绝对定位的 Vue 组件。

#### 其他交互

| 操作 | 行为 |
|---|---|
| 鼠标滚轮 | 平滑缩放，范围 0.15 ~ 3.0，灵敏度适中 |
| 鼠标拖拽画布 | 平移视图 |
| 拖拽节点 | 移动单个节点位置，松手后力导向轻微调整 |
| Hover 节点 | 显示 tooltip：名称、类型、掌握度、所属课程 |
| 键盘 `/` 或 `Ctrl+K` | 激活搜索框 |
| 键盘 `Escape` | 退出聚焦模式 / 关闭右键菜单 / 关闭 AI 面板 |
| 键盘 `Space` | 画布适配（fit view） |

---

## 六、顶部搜索栏

```html
<div class="search-bar">
  <input placeholder="搜索知识点... (Ctrl+K)" />
  <select><!-- 学科筛选 --></select>
</div>
```

**功能：**
1. 前端在图谱数据加载完成后，构建一个本地的节点名称索引（用 `canonicalName` + `name` 字段）
2. 输入时**实时模糊匹配**，显示下拉候选列表（最多 8 个），零延迟
3. 选中候选项后：
   - 图谱自动平移 + 缩放，使目标节点居中
   - 目标节点高亮闪烁 3 次（缩放 + 光晕脉冲）
   - 如果该节点在折叠的 Combo 内，先自动展开所属 Combo
   - 右侧面板更新为该节点详情
4. 学科筛选下拉：可选择只显示某课程的节点（其余变灰），选"全部"恢复

---

## 七、左侧面板（280px 宽，可折叠）

### 7.1 知识体系模块

```
切换视图: [按学科] [按类型]

▾ 机器学习     ████████░░ 80%
  ▸ 监督学习
  ▸ 无监督学习
  ▸ 模型评估
▸ 深度学习     ████░░░░░░ 40%
▸ 数学基础     ██████░░░░ 60%
```

- **按学科视图**：按 `chapterTitle` 分组，每组显示进度条（已掌握/总数）
- **按类型视图**：按 `entityType` 分组（概念 / 方法 / 算法 / 应用 / 主题）
- 点击任意条目 → 图谱画布自动平移聚焦到对应区域
- 进度数据来源：`/api/learning/progress` 接口（已登录时）
- 未登录时进度条隐藏，只显示知识体系结构

### 7.2 个人学习进度模块

```
总进度  ████████░░ 78%

✅ 已掌握    156 个
📖 学习中     23 个
⬜ 未学习     45 个
⚠️ 薄弱点     12 个
⭐ 收藏        8 个

📈 本周学习趋势 [简易折线图/柱状图]
```

- 各分类可点击 → 图谱中对应状态的节点**全部高亮**，其余变灰
- 例如点击"薄弱点" → 所有薄弱节点橙色脉冲高亮
- 数据来源：`/api/dashboard/summary` + `/api/learning/progress`
- 未登录时显示提示"登录后查看个人进度"

### 7.3 学习路径模块

```
🤖 AI 推荐路径
┌─────────────────────────┐
│ 偏导数 → 梯度下降 → 反向 │
│ 传播 → 损失函数 → CNN    │
│          [在图谱中显示]   │
│          [开始学习]       │
└─────────────────────────┘

⭐ 我的路径
▸ NLP入门路径 (6/12)
▸ CV快速上手 (2/8)

＋ 新建自定义路径
```

- **AI 推荐路径**：调用 `/api/recommend/path` 获取
- **自定义路径**：用户通过右键菜单"加入学习路径"添加节点，可拖拽排序
- **"在图谱中显示"**：点击后图谱中路径节点串联高亮，用流光粒子动画连线（粒子沿路径方向流动）
- 数据持久化：自定义路径暂存 localStorage（key: `kg-custom-paths`）

### 7.4 收藏夹模块

- 显示所有已收藏的节点列表
- 点击收藏项 → 图谱聚焦到该节点
- 数据暂存 localStorage（key: `kg-bookmarks`）

---

## 八、右侧详情面板（320px 宽，可折叠）

当用户点击图谱中的知识点节点时，右侧面板**自动滑入更新**（从右侧滑入的动画，300ms ease-out）。

```
┌─ 节点详情 ─────────────────────┐
│                                 │
│  🔵 梯度下降                    │
│  类型: 算法 | 课程: 机器学习     │
│  掌握度: ████░░ 65%             │
│                                 │
│  📋 简介                        │
│  梯度下降是一种迭代优化算法...    │
│                                 │
│  🔗 前置知识                     │
│  • 偏导数 ✅ (已掌握)            │  ← 点击跳转
│  • 损失函数 📖 (学习中)          │
│                                 │
│  🔗 后续可学                     │
│  • Adam优化器                   │
│  • 随机梯度下降                 │
│                                 │
│  🔗 相关知识                     │
│  • 学习率                       │
│  • 反向传播                     │
│                                 │
│  📝 我的笔记                     │
│  "注意学习率不能设太大..."        │
│  [编辑笔记]                      │
│                                 │
│  ────────────────────────────── │
│  [📖 开始学习]  [🤖 AI讲解]     │
│                                 │
└─────────────────────────────────┘
```

**数据获取**：选中节点后调用 `/api/knowledge/:knowledgeId`

**前置/相关/后续知识列表中的每一项**：
- 点击 → 图谱自动聚焦到该节点 + 右侧面板切换到该节点
- 旁边显示学习状态图标（✅ 已掌握 / 📖 学习中 / ⬜ 未学习）

**"开始学习"按钮**：跳转到独立的学习详情页（路由 `/knowledge/:knowledgeId`）

---

## 九、AI 助手

### 9.1 触发方式

右下角悬浮按钮（圆形，60px，带脉冲动画吸引注意力），点击展开为 400×600 的对话面板。

### 9.2 对话面板设计

```
┌─ AI 学习助手 ──────── [✕] ─┐
│                              │
│  (对话消息列表，可滚动)       │
│                              │
│  AI: 你好！我是你的 AI 学伴。 │
│  当前你选中了"梯度下降"。     │
│  有什么我可以帮你的？         │
│                              │
│  💡 快捷问题：               │
│  [帮我解释] [对比分析]       │
│  [推荐下一步] [高亮前置链]   │
│                              │
│  ┌────────────────── [发送]┐ │
│  │ 输入消息...              │ │
│  └──────────────────────── ┘ │
└──────────────────────────────┘
```

### 9.3 功能要求

1. **上下文感知**：自动带入当前选中节点的信息（名称、描述、前置知识、掌握度）作为 system prompt
2. **流式输出**：使用 SSE 流式渲染 AI 回复，打字机效果
3. **图谱操控能力**：AI 回复中如果提到了图谱中存在的知识点名称，自动变为**可点击的链接**，点击后图谱聚焦到该节点
4. **快捷问题**：根据当前选中节点自动生成 4 个快捷问题按钮
5. **对话持久化**：同一节点的对话历史保持在内存中，切换回来可继续
6. **API 对接**：
   - 已登录：调用 `/api/assistant/chat`（POST，SSE 流式）
   - 未登录：调用 `/api/chat`（POST，SSE 流式）

### 9.4 SSE 流式解析

后端返回格式：

```
data: {"choices":[{"delta":{"content":"你好"}}]}
data: {"choices":[{"delta":{"content":"！"}}]}
data: [DONE]
```

解析逻辑：逐行读取 `data: ` 前缀，解析 JSON，提取 `choices[0].delta.content` 拼接。

---

## 十、路由设计

```javascript
const routes = [
  {
    path: '/',
    component: GraphHome,          // 图谱主页（沉浸式全屏）
    meta: { fullscreen: true }
  },
  {
    path: '/knowledge/:knowledgeId',
    component: KnowledgeDetail,    // 知识点深度学习页
  },
  {
    path: '/login',
    component: LoginView,
    meta: { guestOnly: true }
  },
  {
    path: '/register',
    component: RegisterView,
    meta: { guestOnly: true }
  },
]
```

主页就是图谱页面。其他页面（登录、注册、知识详情）作为覆盖层或独立路由。

---

## 十一、Pinia Store 设计

### graphStore

```javascript
// stores/graphStore.js
export const useGraphStore = defineStore('graph', () => {
  // === 数据 ===
  const rawNodes = ref([])          // 后端原始节点
  const rawEdges = ref([])          // 后端原始边
  const loading = ref(false)
  const error = ref('')
  
  // === 图谱状态 ===
  const selectedNodeId = ref(null)     // 当前选中的节点 ID
  const focusedNodeId = ref(null)      // 当前聚焦（高亮邻居）的节点 ID
  const expandedCombos = ref(new Set()) // 已展开的 Combo ID 集合
  const highlightFilter = ref(null)    // 'mastered' | 'learning' | 'weak' | 'bookmarked' | null
  const highlightPath = ref(null)      // 高亮的学习路径节点 ID 数组
  
  // === 计算 ===
  const courseGroups = computed(...)    // 按 chapterTitle 分组
  const nodeIndex = computed(...)      // 用于搜索的 name → id 索引
  
  // === 操作 ===
  async function loadFullGraph() { ... }
  function selectNode(nodeId) { ... }
  function focusNode(nodeId) { ... }
  function toggleCombo(comboId) { ... }
  function clearFocus() { ... }
  function highlightByStatus(status) { ... }
  function highlightPathNodes(nodeIds) { ... }
  function searchNodes(keyword) { ... }
  
  return { ... }
})
```

### userStore

```javascript
// stores/userStore.js
export const useUserStore = defineStore('user', () => {
  const token = ref('')
  const user = ref(null)
  const isAuthenticated = computed(() => Boolean(token.value && user.value))
  
  // 学习进度
  const progressList = ref([])
  const progressMap = computed(...)     // knowledgeId → progress item
  
  // 收藏 & 笔记（暂用 localStorage）
  const bookmarks = ref([])            // routeKnowledgeId[]
  const notes = ref({})                // { [knowledgeId]: string }
  const customPaths = ref([])          // 自定义学习路径
  
  async function login(username, password) { ... }
  async function register(username, password) { ... }
  async function loadProgress() { ... }
  function toggleBookmark(knowledgeId) { ... }
  function saveNote(knowledgeId, content) { ... }
  function addToPath(pathId, knowledgeId) { ... }
  
  return { ... }
})
```

---

## 十二、前端文件结构

```
frontend/
├── index.html
├── package.json
├── vite.config.js
├── src/
│   ├── main.js                          # 入口：创建 app, 挂载 pinia/router
│   ├── App.vue                          # 根组件
│   ├── router.js                        # 路由配置
│   ├── style.css                        # 全局样式（深色主题变量、reset）
│   │
│   ├── api/                             # API 层
│   │   ├── request.js                   # 封装 fetch + token + 错误处理
│   │   ├── graph.js                     # 图谱相关 API
│   │   ├── auth.js                      # 登录注册
│   │   ├── learning.js                  # 学习进度、题目
│   │   ├── recommend.js                 # 推荐路径
│   │   └── assistant.js                 # AI 助手
│   │
│   ├── stores/                          # Pinia stores
│   │   ├── graphStore.js
│   │   └── userStore.js
│   │
│   ├── views/                           # 页面级组件
│   │   ├── GraphHome.vue                # 主页：图谱 + 侧边栏 + 搜索
│   │   ├── KnowledgeDetail.vue          # 知识点深度学习页
│   │   ├── LoginView.vue
│   │   └── RegisterView.vue
│   │
│   ├── components/                      # 可复用组件
│   │   ├── graph/
│   │   │   ├── GraphCanvas.vue          # G6 画布核心组件
│   │   │   ├── GraphControls.vue        # 缩放/全屏/图例/小地图 控件
│   │   │   ├── ContextMenu.vue          # 右键菜单
│   │   │   └── NodeTooltip.vue          # hover tooltip
│   │   │
│   │   ├── sidebar/
│   │   │   ├── LeftPanel.vue            # 左侧面板容器（可折叠）
│   │   │   ├── KnowledgeTree.vue        # 知识体系树
│   │   │   ├── ProgressPanel.vue        # 学习进度面板
│   │   │   ├── PathPanel.vue            # 学习路径面板
│   │   │   └── BookmarkPanel.vue        # 收藏夹面板
│   │   │
│   │   ├── detail/
│   │   │   └── RightPanel.vue           # 右侧详情面板（随选中更新）
│   │   │
│   │   ├── search/
│   │   │   └── SearchBar.vue            # 顶部搜索
│   │   │
│   │   └── ai/
│   │       └── AiAssistant.vue          # AI 助手悬浮面板
│   │
│   ├── composables/                     # 可复用逻辑
│   │   ├── useGraph.js                  # G6 实例管理、数据转换
│   │   ├── useNodeFocus.js              # 聚焦/高亮/动画逻辑
│   │   ├── useContextMenu.js            # 右键菜单状态
│   │   ├── useAiChat.js                 # AI 对话逻辑（SSE 流式）
│   │   └── useKatex.js                  # LaTeX 渲染
│   │
│   └── utils/
│       ├── graphConfig.js               # G6 节点/边/Combo 自定义配置
│       ├── graphTransform.js            # 后端数据 → G6 数据格式转换
│       ├── colorMap.js                  # 课程→颜色映射表 + 分配逻辑
│       └── searchIndex.js              # 前缀匹配搜索索引
```

---

## 十三、CSS 主题变量

```css
:root {
  /* 背景 */
  --bg-primary: #081018;
  --bg-secondary: #0F1923;
  --bg-panel: rgba(15, 25, 35, 0.92);
  --bg-hover: rgba(255, 255, 255, 0.06);
  --bg-active: rgba(255, 255, 255, 0.1);
  
  /* 文字 */
  --text-primary: rgba(241, 245, 249, 0.94);
  --text-secondary: rgba(226, 232, 240, 0.72);
  --text-muted: rgba(148, 163, 184, 0.5);
  
  /* 边框 */
  --border-subtle: rgba(255, 255, 255, 0.08);
  --border-medium: rgba(255, 255, 255, 0.15);
  
  /* 圆角 */
  --radius-sm: 8px;
  --radius-md: 12px;
  --radius-lg: 16px;
  --radius-full: 999px;
  
  /* 阴影 */
  --shadow-panel: 0 16px 48px rgba(0, 0, 0, 0.3);
  
  /* 面板尺寸 */
  --left-panel-width: 280px;
  --right-panel-width: 320px;
  --topbar-height: 56px;
  
  /* 动画 */
  --transition-fast: 150ms ease;
  --transition-normal: 300ms ease-out;
}
```

---

## 十四、Vite 配置

```javascript
// vite.config.js
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'esnext',
  },
})
```

---

## 十五、性能优化要求

1. **首屏只渲染 Combo 节点**：初始状态所有课程折叠，页面只有 ~20 个 Combo 节点，秒开
2. **WebGL 渲染器**：G6 v5 必须开启 WebGL renderer
3. **展开时增量添加**：展开 Combo 时只添加该课程的节点，不重绘整图
4. **节点文字 LOD**：缩放级别低于 0.5 时隐藏所有文字标签，只显示圆点
5. **搜索零延迟**：前端维护内存中的 name 索引，不走网络请求
6. **数据缓存**：已加载的子图、知识点详情缓存在 Pinia store 中
7. **图片/图标懒加载**：面板内的图片按需加载

---

## 十六、额外增强特性

### 16.1 学习路径流光动画

当用户选择一条学习路径高亮时，路径上的节点之间用**流光粒子动画**串联：
- 粒子沿路径方向流动（从前置到后续）
- 粒子颜色渐变：起点蓝色 → 终点金色
- 像"电流流过电路"或"神经信号传导"
- 实现方式：G6 v5 的 edge 动画 + 自定义 shader 或 Canvas 粒子

### 16.2 成就徽章

- 当某课程所有知识点掌握度 >= 80% 时，该课程 Combo 外圈变为**金色发光**边框
- 首次达成时播放一次"星星爆炸"粒子动画
- Combo 标签增加 "⭐ 已精通" 文字

### 16.3 小地图

- 右下角显示整个图谱的缩略鸟瞰图（200×150px）
- 矩形框标注当前视口位置
- 可拖拽矩形框快速导航
- 使用 G6 v5 内置 Minimap 插件

### 16.4 键盘快捷键

| 快捷键 | 功能 |
|---|---|
| `Ctrl+K` 或 `/` | 聚焦搜索框 |
| `Escape` | 退出聚焦 / 关闭菜单 / 关闭面板 |
| `Space` | 画布适配（fit view） |
| `[` | 折叠左侧面板 |
| `]` | 折叠右侧面板 |

---

## 十七、响应式适配

- 宽度 < 768px 时：左右面板默认折叠，通过底部 Tab 切换
- 图谱画布始终全屏，面板以抽屉形式覆盖

---

## 十八、实现顺序建议

请按以下顺序实现，每完成一步确保可运行：

1. **项目脚手架**：Vite + Vue 3 + Pinia + vue-router + 全局 CSS 变量 + 深色主题
2. **API 层**：封装 fetch request，实现所有 API 函数
3. **Pinia stores**：graphStore + userStore，实现数据加载逻辑
4. **GraphCanvas 组件**：G6 v5 初始化，加载全量数据，渲染所有节点和边（先不做 Combo，先做 flat 布局确认数据正确）
5. **Combo 分层**：实现数据转换（按 chapterTitle 分组）、Combo 展开/折叠
6. **节点交互**：左键聚焦、右键菜单、hover tooltip
7. **节点视觉**：课程颜色、光晕效果、呼吸动画、线条粗细
8. **搜索栏**：本地模糊搜索 + 自动聚焦动画
9. **右侧详情面板**：选中节点 → 加载详情 → 面板滑入
10. **左侧面板**：知识体系 + 学习进度 + 学习路径 + 收藏夹
11. **AI 助手**：悬浮按钮 + 对话面板 + SSE 流式 + 上下文感知
12. **增强特性**：流光动画、成就徽章、小地图、键盘快捷键
13. **登录/注册**：对接已有 API
14. **知识详情页**：独立的深度学习页面
15. **响应式适配**

---

## 十九、最终验收标准

- [ ] 打开网站首屏即看到知识图谱，课程 Combo 默认折叠
- [ ] 点击 Combo 可展开看到内部知识点，动画流畅
- [ ] 缩放、拖拽、移动操作丝滑无卡顿（1000+ 节点下 ≥ 30fps）
- [ ] 不同课程用不同颜色区分
- [ ] 关系线条用中文标签（前置知识/属于/相关）
- [ ] 已掌握节点有金色光晕呼吸动画
- [ ] 左键点击聚焦：只高亮相关，其余变灰
- [ ] 右键菜单：加入路径、收藏、笔记、AI 讲解等
- [ ] 左侧面板：知识体系/进度/路径/收藏全部可用
- [ ] 右侧面板：随节点选中自动更新
- [ ] 搜索框：实时匹配 + 自动聚焦到目标节点
- [ ] AI 助手：流式对话、上下文感知当前节点
- [ ] 收藏和笔记可持久化（localStorage），刷新后仍在
- [ ] 学习路径可保存，可在图谱上高亮显示
