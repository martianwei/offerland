CREATE TABLE IF NOT EXISTS majors (
    major_id uuid NOT NULL PRIMARY KEY,
    major_name VARCHAR(255) NOT NULL,
    school_id uuid NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
    degree_id uuid NOT NULL REFERENCES degrees(degree_id) ON DELETE CASCADE,
    department_id uuid NOT NULL REFERENCES departments(department_id) ON DELETE CASCADE
);
