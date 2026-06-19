CREATE TABLE IF NOT EXISTS student_learning_profiles (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    summary VARCHAR(1000) NOT NULL DEFAULT '',
    source_text TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_student_learning_profiles_student (student_id)
);

CREATE TABLE IF NOT EXISTS student_profile_dimensions (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    profile_id BIGINT UNSIGNED NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    dimension_key VARCHAR(64) NOT NULL,
    dimension_name VARCHAR(128) NOT NULL,
    dimension_value VARCHAR(500) NOT NULL,
    confidence DECIMAL(5,2) NOT NULL DEFAULT 0,
    evidence VARCHAR(500) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_profile_dimensions_student_key (student_id, dimension_key),
    KEY idx_student_profile_dimensions_profile (profile_id),
    KEY idx_student_profile_dimensions_key (dimension_key)
);

CREATE TABLE IF NOT EXISTS student_profile_versions (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    profile_id BIGINT UNSIGNED NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    summary VARCHAR(1000) NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_student_profile_versions_profile (profile_id),
    KEY idx_student_profile_versions_student (student_id)
);

ALTER TABLE student_learning_profiles
    ADD CONSTRAINT fk_student_learning_profiles_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE student_profile_dimensions
    ADD CONSTRAINT fk_student_profile_dimensions_profile_id FOREIGN KEY (profile_id) REFERENCES student_learning_profiles(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_student_profile_dimensions_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE student_profile_versions
    ADD CONSTRAINT fk_student_profile_versions_profile_id FOREIGN KEY (profile_id) REFERENCES student_learning_profiles(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_student_profile_versions_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;
