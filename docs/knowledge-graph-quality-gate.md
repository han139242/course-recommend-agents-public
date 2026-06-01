# 知识图谱质量门禁

本文档说明课程知识图谱在新增或调整批次前后的质量检查流程。目标是让图谱既覆盖全面，又能保持节点定义、关系方向和跨课程连接的一致性。

## 门禁目标

- 课程覆盖：`course_expansion_plan.json` 中登记的 20 门课程都必须有对应 curated batch。
- 节点覆盖：课程节点数必须落在计划区间内，不能靠少量节点撑页面展示。
- Taxonomy 覆盖：`reference_taxonomy.json` 中的参考主题必须全部能在 curated 图谱节点中找到。
- 类型合法：课程批次中的 `entity_type` 必须属于 `topic / concept / method / algorithm / application / model / tool / framework`；`exercise` 只用于练习或非课程型数据。
- 定义一致：同一个 `canonical_name` 可以跨课程复用，但 `entity_type` 和 `description` 不能互相冲突。
- 关系正确：关系类型必须来自白名单，关系两端必须都有节点定义，不能有自环。
- 结构连通：课程图谱不能出现孤立节点，整体应形成可遍历的一张网。
- 先修方向：`PREREQUISITE_OF` 必须表示“先学 source，再学 target”，脚本会对明显反向的边给出人工复核警告。

## 运行方式

```bash
cd pipelines/txt2neo4j
$env:PYTHONPATH = "src"
python -m txt2neo4j.graph_audit
```

安装为本地包后也可以运行：

```bash
cd pipelines/txt2neo4j
txt2neo4j-graph-audit
```

默认输出：

- `pipelines/txt2neo4j/artifacts/graph_quality/knowledge_graph_audit.json`
- `pipelines/txt2neo4j/artifacts/graph_quality/knowledge_graph_audit.md`

如需在 CI 或赛前验收中把警告也视作失败：

```bash
python -m txt2neo4j.graph_audit --fail-on-warning
```

## 新增知识批次流程

1. 在 `config/curated_batches/` 新增或修改 JSON 批次。
2. 保证每个节点都有稳定的 `canonical_name`、合法 `entity_type` 和可解释的 `description`。
3. 对跨课程复用节点，直接复用已有 `canonical_name`，不要写成近义变体。
4. 使用 `PREREQUISITE_OF` 时按“前置知识 -> 后续知识”的方向写。
5. 运行 `python -m txt2neo4j.graph_audit`，只有 `gate_status = pass` 才进入 Neo4j 导入。
6. 导入后再运行 Neo4j 侧验证脚本，确认生产图状态与离线源数据一致。

## 当前验收口径

当前图谱审计使用课程计划、参考 taxonomy、curated seed 和全部 curated batch 作为事实源。审计通过时，可以证明：

- 20 门课程均被覆盖，包括数学与编程基础、AI 核心课、应用方向、工程治理方向，以及 AI 基础设施、高级大模型应用、AI 安全、AI for Science、具身智能与世界模型等前沿扩展课程。
- 参考 taxonomy 主题全部覆盖。
- 同名跨课程节点没有定义冲突。
- 图谱没有孤立节点。
- 整体图谱弱连通，学生可以从基础知识一路跳转到应用和前沿主题。

## 学生学习闭环补强

`batch_33_student_learning_support_bridge.json` 用于补齐学生端教学平台最关键的学习闭环语义，不替代 20 门课程本体，而是把已有的智能教育、学习分析、知识追踪、认知诊断、知识点推荐、题目推荐、学习路径规划、自适应学习、知识图谱和大语言模型等节点串成可解释的学习支持网络。

该批次新增的核心能力包括：

- 课程知识图谱、课程能力图谱和先修链路推理。
- 学习目标建模、薄弱点识别、知识点难度估计和多源学习证据融合。
- 学习路径推荐、个性化学习路径、学习资源匹配和复习间隔调度。
- 错题诊断、练习反馈、学习解释生成和学习闭环评估。

这组节点直接服务学生端的“我现在该学什么、为什么这样学、错在哪里、下一步怎么补”四个问题，避免图谱只停留在课程目录展示层。

## AI 专业知识域补全

`batch_34_ai_professional_body_of_knowledge_completion.json` 用于对齐 ACM/IEEE-CS/AAAI CS2023 中人工智能知识域的专业覆盖要求，补充已有课程图谱中尚未显式出现的标准知识点。该批次不是替代机器学习、自然语言处理、计算机视觉、强化学习和大语言模型等课程文件，而是补齐这些课程之间的专业主干和跨域桥接节点。

覆盖范围包括：
- 人工智能基础问题：智能定义、强/弱人工智能、理性行为、任务环境属性和问题形式化。
- 搜索与约束求解：状态空间、启发式搜索、A* 系列、局部搜索、约束传播和博弈中的随机机会节点。
- 知识表示与推理：一阶逻辑、归结、链式推理、非单调推理、时空推理、本体工程和知识一致性检查。
- 概率推理与决策：贝叶斯决策理论、d-分离、马尔可夫毯、变量消除、采样推理和影响图。
- 自动规划：STRIPS、PDDL、规划图、GraphPlan、HTN、条件规划、概率规划、MDP 规划和动态重规划。
- 智能体：智能体架构、认知架构、BDI、决策理论智能体、协作智能体、可信智能体和智能体工作流。
- NLP、机器人与感知：形式文法、句法分析、语义表示、SLAM、导航、机械臂操作、多模态感知、三维重建和场景理解。
- 机器学习理论与可信 AI：假设空间、归纳偏置、泛化误差、结构风险最小化、偏差、公平性、鲁棒性、安全性、隐私保护、元学习、持续学习、贝叶斯学习、因果学习和逆强化学习。

新增或调整该批次时仍需满足同一套门禁：不重复定义已有 `canonical_name`，关系端点必须存在，`PREREQUISITE_OF` 保持“先学 source，再学 target”的方向，且全图审计必须达到 `gate_status = pass`。

`batch_35_ai_professional_deep_extension.json` 进一步补充教材和工业项目中常见但容易遗漏的细粒度知识点，重点覆盖：
- 任务环境与智能体类型：静态/动态环境、反射型/目标型/效用型智能体、半自主、人类在环和具身智能体。
- 搜索内部机制：问题状态、目标测试、搜索操作符、搜索前沿、闭集、复杂度、局部极值、博弈搜索截断和开局/残局知识。
- 知识工程与语义系统：符号/逻辑/子符号表示、知识组织、知识利用、开放世界/闭合世界假设、本体对齐与映射。
- 概率推断：马尔可夫链、贝叶斯滤波、因子图、团树、变分推断、期望传播、前向后向、维特比和卡尔曼滤波扩展。
- 规划执行：动作前提/效果、偏序规划约束、计划修复、在线规划、时态动作、资源模型、SATPlan 和 POMDP 规划。
- 机器人与感知：激光雷达、深度相机、事件相机、触觉传感器、齐次变换、运动学链、拓扑/语义地图、避障和控制方法。
- 经典视觉/NLP：低层视觉、相机标定、RANSAC、霍夫变换、纹理分析、词法分析、文法、语用、NLI 和对话策略。
- 学习理论、生成模型与安全治理：PAC、VC 维、泛化界、Rademacher 复杂度、双下降、域自适应、生成模型、模型卡/数据卡、越狱、后门、隐私泄露和风险治理。

为避免“字符串不同但语义相同”的工业级脏数据，`graph_audit` 现在会读取 `dictionaries/term_dictionary.json`，如果某个别名又被作为独立 `canonical_name` 定义，会以 `dictionary.alias_canonical_collision` 直接判为错误。例如 `A*搜索` 应复用 `A*搜索算法`，不能另建同义节点。
