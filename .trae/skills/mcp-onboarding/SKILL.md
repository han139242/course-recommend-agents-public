---
name: "mcp-onboarding"
description: "为本项目选择并启用合适的 MCP 工具组合。用户说“配置mcp/想用浏览器或Docker或数据库工具”时调用。"
---

# MCP Onboarding

## 现状

- 当前环境已启用：`integrated_browser`（用于网页/前端联调、抓取页面结构、截图、网络请求排查）

## 推荐 MCP 组合（按你们项目架构图）

### A. 开发必备（最小集）

- Browser：`integrated_browser`（已启用）
- GitHub：用于 Issues/PR/文件浏览与发布（团队协作更顺）
- FileSystem：用于批量整理文件、生成目录结构（减少手工操作）

### B. 数据库联调（推荐）

- Docker：用于启动/停止 MySQL/Neo4j/Redis（与 `docker-compose.yml` 配套）
- MySQL：用于跑迁移、检查表结构、执行查询
- Neo4j：用于导入/导出、跑 Cypher、检查图结构

### C. 数据抽取与爬虫（可选）

- HTTP：用于请求调试、重放抓取接口
- Scheduler/Queue：用于离线任务编排（看团队是否需要）

## 选择建议

- 你们如果要“所有组员直接连同一套数据库”：优先选 B（数据库联调）
- 你们如果先做前端演示：优先选 A（开发必备）
- 你们如果重点做离线抽取：再加 C（数据抽取与爬虫）

## 团队落地（你当前选择：D 全都要）

- 每个组员在自己的 Trae 环境里分别启用/安装对应 MCP（MCP 属于本机能力，通常不跟随仓库自动安装）
- 最小要求：
  - Browser：已启用 `integrated_browser`
  - GitHub：建议启用一个 GitHub MCP（能查 issues/pr、读写仓库文件等）
  - FileSystem：建议启用一个文件系统 MCP（用于批量整理、生成目录等）
- 数据库联调：
  - Docker：建议启用一个 Docker MCP（用于拉起/查看容器状态）
  - MySQL：建议启用一个 MySQL MCP（用于查询/迁移/检查表结构）
  - Neo4j：建议启用一个 Neo4j MCP（用于执行 Cypher、检查图结构）
- 抽取与爬虫：
  - HTTP：建议启用一个 HTTP/Fetch 类 MCP（用于调试接口请求）
