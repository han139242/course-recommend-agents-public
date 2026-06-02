CREATE TABLE IF NOT EXISTS student_goal_profiles (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    grade_level VARCHAR(32) NOT NULL,
    major VARCHAR(128) NOT NULL DEFAULT '',
    goal_type VARCHAR(32) NOT NULL,
    goal_text VARCHAR(500) NOT NULL,
    target_direction VARCHAR(255) NOT NULL DEFAULT '',
    learned_topics_json TEXT NOT NULL,
    foundation_level VARCHAR(32) NOT NULL,
    weekly_hours INT NOT NULL DEFAULT 6,
    learning_preference VARCHAR(32) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_student_goal_profiles_student (student_id),
    KEY idx_student_goal_profiles_goal_type (goal_type),
    KEY idx_student_goal_profiles_grade_level (grade_level)
);

CREATE TABLE IF NOT EXISTS student_study_plans (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    student_id BIGINT UNSIGNED NOT NULL,
    profile_id BIGINT UNSIGNED NOT NULL,
    summary VARCHAR(500) NOT NULL,
    stages_json TEXT NOT NULL,
    next_steps_json TEXT NOT NULL,
    generated_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_student_study_plans_student (student_id),
    KEY idx_student_study_plans_profile (profile_id),
    KEY idx_student_study_plans_generated_at (generated_at)
);

ALTER TABLE student_goal_profiles
    ADD CONSTRAINT fk_student_goal_profiles_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE student_study_plans
    ADD CONSTRAINT fk_student_study_plans_student_id FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_student_study_plans_profile_id FOREIGN KEY (profile_id) REFERENCES student_goal_profiles(id) ON DELETE RESTRICT;
