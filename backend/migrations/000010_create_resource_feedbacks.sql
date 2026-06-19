CREATE TABLE IF NOT EXISTS resource_feedbacks (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    resource_id BIGINT UNSIGNED NOT NULL,
    job_id BIGINT UNSIGNED NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    rating INT NOT NULL DEFAULT 0,
    difficulty VARCHAR(32) NOT NULL DEFAULT '',
    helpful BOOLEAN NOT NULL DEFAULT FALSE,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    comment VARCHAR(500) NOT NULL DEFAULT '',
    next_action VARCHAR(500) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_resource_feedback_student_resource (resource_id, student_id),
    KEY idx_resource_feedback_job (job_id),
    KEY idx_resource_feedback_student (student_id),
    KEY idx_resource_feedback_rating (rating),
    KEY idx_resource_feedback_updated_at (updated_at)
);

ALTER TABLE resource_feedbacks
    ADD CONSTRAINT fk_resource_feedbacks_resource_id FOREIGN KEY (resource_id) REFERENCES generated_resources(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_resource_feedbacks_job_id FOREIGN KEY (job_id) REFERENCES resource_generation_jobs(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_resource_feedbacks_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;
