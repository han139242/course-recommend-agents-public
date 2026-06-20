CREATE TABLE IF NOT EXISTS learning_events (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    knowledge_id VARCHAR(128) NOT NULL DEFAULT '',
    knowledge_name VARCHAR(255) NOT NULL DEFAULT '',
    resource_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    resource_type VARCHAR(64) NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_learning_events_student_id (student_id),
    KEY idx_learning_events_type (event_type),
    KEY idx_learning_events_knowledge_id (knowledge_id),
    KEY idx_learning_events_resource_id (resource_id),
    KEY idx_learning_events_occurred_at (occurred_at)
);

CREATE TABLE IF NOT EXISTS resource_push_items (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    resource_id BIGINT UNSIGNED NOT NULL,
    job_id BIGINT UNSIGNED NOT NULL,
    knowledge_id VARCHAR(128) NOT NULL DEFAULT '',
    knowledge_name VARCHAR(255) NOT NULL DEFAULT '',
    course_title VARCHAR(255) NOT NULL DEFAULT '',
    stage VARCHAR(64) NOT NULL DEFAULT 'today',
    reason VARCHAR(500) NOT NULL DEFAULT '',
    priority INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_resource_push_items_student_id (student_id),
    KEY idx_resource_push_items_resource_id (resource_id),
    KEY idx_resource_push_items_job_id (job_id),
    KEY idx_resource_push_items_knowledge_id (knowledge_id),
    KEY idx_resource_push_items_priority (priority),
    KEY idx_resource_push_items_status (status)
);

ALTER TABLE learning_events
    ADD CONSTRAINT fk_learning_events_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE resource_push_items
    ADD CONSTRAINT fk_resource_push_items_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_resource_push_items_resource_id FOREIGN KEY (resource_id) REFERENCES generated_resources(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_resource_push_items_job_id FOREIGN KEY (job_id) REFERENCES resource_generation_jobs(id) ON DELETE CASCADE;
