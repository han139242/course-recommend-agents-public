CREATE TABLE IF NOT EXISTS learning_progress (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT NOT NULL,
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
