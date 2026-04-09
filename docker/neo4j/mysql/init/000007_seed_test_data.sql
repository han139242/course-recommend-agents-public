-- Docker 本地联调种子数据：测试账号与示例题目（明文密码均为 TestPass123! 对应的 bcrypt 哈希）。
SET NAMES utf8mb4;

INSERT INTO users (id, name, email, password_hash, role, title, student_no, class_id, target, status, created_by)
VALUES
  (1, '测试教师', 'teacher@test.local', '$2a$10$TBpy3melDS4.eeqAqb8LfO6OMTvfAWleU35l94krb/hGZAWv3YgFO', 'teacher', '讲师', NULL, NULL, '', 'active', NULL);

INSERT INTO classrooms (id, name, grade, teacher_id, status)
VALUES (1, 'Docker 测试班', '本科一年级', 1, 'active');

INSERT INTO users (id, name, email, password_hash, role, title, student_no, class_id, target, status, created_by)
VALUES
  (2, '测试学生', 'student@test.local', '$2a$10$TBpy3melDS4.eeqAqb8LfO6OMTvfAWleU35l94krb/hGZAWv3YgFO', 'student', NULL, 'S20260001', 1, '掌握人工智能基础', 'active', 1);

INSERT INTO questions (id, question_code, type, knowledge_id, stem, correct_option, difficulty, explanation, status, created_by)
VALUES
  (1, 'Q-DEMO-001', 'single_choice', '002edb92a1990f62', '「会话记忆」在多轮对话中的主要作用更接近下列哪一项？', 'A', 0.50, '对应知识图谱节点：会话记忆。', 'active', 1);

INSERT INTO question_options (question_id, option_key, option_text, sort_order) VALUES
  (1, 'A', '在多轮交互中保留历史上下文与用户偏好', 0),
  (1, 'B', '压缩模型体积以降低显存占用', 1),
  (1, 'C', '对权重矩阵做结构化剪枝', 2),
  (1, 'D', '对传输链路做端到端加密', 3);
