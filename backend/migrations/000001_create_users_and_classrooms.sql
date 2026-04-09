CREATE TABLE IF NOT EXISTS users (
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

CREATE TABLE IF NOT EXISTS classrooms (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    grade VARCHAR(64) NULL,
    teacher_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_classrooms_teacher_id (teacher_id),
    KEY idx_classrooms_status (status)
);

ALTER TABLE classrooms
    ADD CONSTRAINT fk_classrooms_teacher_id FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE users
    ADD CONSTRAINT fk_users_class_id FOREIGN KEY (class_id) REFERENCES classrooms(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
