开发环境与生产环境都应优先显式执行 migrations 目录下的 SQL 脚本创建表结构。

当前已提供：
- 000001_create_users_and_classrooms.sql

说明：
- 生产环境不依赖应用启动时自动建表。
- seed 只负责写入开发演示数据，不负责建表。
- 执行 seed 前，请先确保 migration 已经执行完成。
