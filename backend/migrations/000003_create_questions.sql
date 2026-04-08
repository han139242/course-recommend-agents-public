CREATE TABLE IF NOT EXISTS questions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    question_code VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,
    knowledge_id VARCHAR(64) NOT NULL,
    stem TEXT NOT NULL,
    correct_option VARCHAR(8) NOT NULL,
    difficulty DECIMAL(4,2) NOT NULL,
    explanation TEXT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_by BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_questions_question_code (question_code),
    KEY idx_questions_knowledge_id (knowledge_id),
    KEY idx_questions_status (status),
    KEY idx_questions_created_by (created_by)
);

ALTER TABLE questions
    ADD CONSTRAINT fk_questions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT;
