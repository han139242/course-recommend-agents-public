# 课程推荐智能体 — Wiki 文档导航

本 Wiki 包含系统的核心设计文档，供团队开发参考。

## 文档索引

| 文档 | 说明 |
|---|---|
| [设计文档](设计文档) | 全局架构设计、模块划分、存储规约、数据流转闭环 |
| [数据库设计文档](数据库设计文档) | MySQL 表结构、Redis 键设计、Neo4j 图模型 |
| [在线 API 契约](online-api-contract) | 前端 → Go API Server 的完整 HTTP 接口定义 |
| [前沿知识更新链路设计](frontier-knowledge-pipeline) | 课后扩展与前沿论文接入的数据模型与方案 |

## 技术栈

- **前端**：Vue 3 + Vite + ECharts + D3.js
- **在线服务**：Go（单体部署，内部模块化）
- **离线流水线**：Python（爬虫、语料处理、LLM 抽取、图谱写入）
- **基础设施**：Neo4j / MySQL / Redis
