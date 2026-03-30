---
name: "db-bootstrap"
description: "生成/维护 MySQL+Neo4j(可选Redis)的一键部署与连接配置。用户要“连数据库/团队共享数据库/部署到云服务器”时调用。"
---

# DB Bootstrap

## 目标

- 提供可共享的数据库连接配置模板（不泄露真实密码）
- 提供可复现的一键启动方案（Docker Compose）
- 给出最小的连通性检查步骤（能连上、能执行简单查询）

## 推荐交付物（仓库内）

- `.env.example`: 连接配置模板
- `.gitignore`: 忽略 `.env` 等敏感文件
- `docker-compose.yml`: MySQL + Neo4j（可选 Redis）本地/服务器部署
- `.gitattributes`: 对 `.pdf/.docx/.zip/.bin` 等大文件启用 Git LFS（按需）

## 安全约束

- 任何真实密码只放在 `.env`，永远不要提交到 Git
- 线上数据库必须开启最小权限账号（不要给 root/neo4j 管理员密码到应用）
- 如需公网访问：只开放必要端口，并限制来源 IP（安全组/防火墙）

## 快速验证清单

- MySQL：能连接并执行 `SELECT 1`
- Neo4j：能连接并执行 `RETURN 1`
-（可选）Redis：能连接并 `PING`
