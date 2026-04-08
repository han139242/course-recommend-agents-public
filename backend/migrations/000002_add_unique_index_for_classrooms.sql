ALTER TABLE classrooms
    ADD UNIQUE KEY uk_classrooms_teacher_name (teacher_id, name);
