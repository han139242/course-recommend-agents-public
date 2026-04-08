CREATE TABLE IF NOT EXISTS student_answers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT NOT NULL,
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
