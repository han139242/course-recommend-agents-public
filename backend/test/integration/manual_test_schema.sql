CREATE TABLE users (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL,
    email VARCHAR(128) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(16) NOT NULL,
    title VARCHAR(64) NULL,
    student_no VARCHAR(64) NULL,
    class_id BIGINT UNSIGNED NULL,
    target VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_users_email (email),
    UNIQUE KEY uk_users_student_no (student_no),
    KEY idx_users_role (role),
    KEY idx_users_status (status),
    KEY idx_users_class_id (class_id),
    KEY idx_users_created_by (created_by)
);

CREATE TABLE classrooms (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    grade VARCHAR(64) NULL,
    teacher_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_classrooms_teacher_id (teacher_id),
    KEY idx_classrooms_status (status),
    UNIQUE KEY uk_classrooms_teacher_name (teacher_id, name)
);

ALTER TABLE classrooms
    ADD CONSTRAINT fk_classrooms_teacher_id FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE users
    ADD CONSTRAINT fk_users_class_id FOREIGN KEY (class_id) REFERENCES classrooms(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

CREATE TABLE questions (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    question_code VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,
    knowledge_id VARCHAR(64) NOT NULL,
    stem TEXT NOT NULL,
    correct_option VARCHAR(8) NOT NULL,
    difficulty DECIMAL(4,2) NOT NULL,
    explanation TEXT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_questions_question_code (question_code),
    KEY idx_questions_knowledge_id (knowledge_id),
    KEY idx_questions_status (status),
    KEY idx_questions_created_by (created_by)
);

ALTER TABLE questions
    ADD CONSTRAINT fk_questions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT;

CREATE TABLE question_options (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    question_id BIGINT UNSIGNED NOT NULL,
    option_key VARCHAR(8) NOT NULL,
    option_text TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_question_options_question_option (question_id, option_key),
    KEY idx_question_options_question_id (question_id)
);

ALTER TABLE question_options
    ADD CONSTRAINT fk_question_options_question_id FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE;

CREATE TABLE student_answers (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    question_code VARCHAR(64) NOT NULL,
    knowledge_id VARCHAR(64) NOT NULL,
    selected_option VARCHAR(8) NOT NULL,
    is_correct TINYINT(1) NOT NULL,
    duration_ms INT NULL,
    answered_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_student_answers_student_answered (student_id, answered_at),
    KEY idx_student_answers_student_knowledge (student_id, knowledge_id),
    KEY idx_student_answers_question_answered (question_code, answered_at)
);

ALTER TABLE student_answers
    ADD CONSTRAINT fk_student_answers_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;

CREATE TABLE learning_progress (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    knowledge_id VARCHAR(64) NOT NULL,
    canonical_name VARCHAR(255) NOT NULL,
    mastery_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'not_started',
    correct_rate DECIMAL(5,4) NOT NULL DEFAULT 0,
    answered_count INT NOT NULL DEFAULT 0,
    last_answered_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_learning_progress_student_knowledge (student_id, knowledge_id)
);

ALTER TABLE learning_progress
    ADD CONSTRAINT fk_learning_progress_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;
