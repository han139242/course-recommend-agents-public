CREATE TABLE IF NOT EXISTS resource_generation_jobs (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    knowledge_id VARCHAR(128) NOT NULL DEFAULT '',
    knowledge_name VARCHAR(255) NOT NULL DEFAULT '',
    course_title VARCHAR(255) NOT NULL DEFAULT '',
    target VARCHAR(500) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    progress INT NOT NULL DEFAULT 0,
    current_agent VARCHAR(128) NOT NULL DEFAULT '',
    error_message VARCHAR(1000) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_resource_generation_jobs_student (student_id),
    KEY idx_resource_generation_jobs_status (status),
    KEY idx_resource_generation_jobs_knowledge (knowledge_id),
    KEY idx_resource_generation_jobs_created_at (created_at)
);

CREATE TABLE IF NOT EXISTS resource_agent_steps (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    job_id BIGINT UNSIGNED NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    agent_key VARCHAR(64) NOT NULL,
    agent_name VARCHAR(128) NOT NULL,
    role VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'completed',
    summary VARCHAR(1000) NOT NULL DEFAULT '',
    started_at DATETIME NOT NULL,
    ended_at DATETIME NULL,
    KEY idx_resource_agent_steps_job (job_id),
    KEY idx_resource_agent_steps_student (student_id)
);

CREATE TABLE IF NOT EXISTS generated_resources (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    job_id BIGINT UNSIGNED NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    knowledge_id VARCHAR(128) NOT NULL DEFAULT '',
    knowledge_name VARCHAR(255) NOT NULL DEFAULT '',
    course_title VARCHAR(255) NOT NULL DEFAULT '',
    resource_type VARCHAR(64) NOT NULL,
    type_label VARCHAR(128) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content LONGTEXT NOT NULL,
    format VARCHAR(32) NOT NULL DEFAULT 'markdown',
    reason VARCHAR(500) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'ready',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_generated_resources_job (job_id),
    KEY idx_generated_resources_student (student_id),
    KEY idx_generated_resources_knowledge (knowledge_id),
    KEY idx_generated_resources_type (resource_type),
    KEY idx_generated_resources_created_at (created_at)
);

ALTER TABLE resource_generation_jobs
    ADD CONSTRAINT fk_resource_generation_jobs_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE resource_agent_steps
    ADD CONSTRAINT fk_resource_agent_steps_job_id FOREIGN KEY (job_id) REFERENCES resource_generation_jobs(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_resource_agent_steps_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE generated_resources
    ADD CONSTRAINT fk_generated_resources_job_id FOREIGN KEY (job_id) REFERENCES resource_generation_jobs(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_generated_resources_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;
