开发环境与生产环境都应优先显式执行 migrations 目录下的 SQL 脚本创建表结构。

当前已提供：
- 000001_create_users_and_classrooms.sql
- 000002_add_unique_index_for_classrooms.sql
- 000003_create_questions.sql
- 000004_create_question_options.sql
- 000005_create_student_answers.sql
- 000006_create_learning_progress.sql
- 000007_create_student_study_plan.sql
- 000008_create_learning_profiles.sql
- 000009_create_generated_resources.sql
- 000010_create_resource_feedbacks.sql
- 000011_create_learning_events_and_resource_push_items.sql

说明：
- 生产环境不依赖应用启动时自动建表。
- seed 只负责写入开发演示数据，不负责建表。
- 执行 seed 前，请先确保 migration 已经执行完成。
