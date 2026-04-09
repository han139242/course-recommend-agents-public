CREATE TABLE IF NOT EXISTS question_options (
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
