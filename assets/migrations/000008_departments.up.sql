CREATE TABLE IF NOT EXISTS departments (
    department_id uuid NOT NULL PRIMARY KEY,
    department_name VARCHAR(255) NOT NULL,
    school_id uuid NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
    degree_id uuid NOT NULL REFERENCES degrees(degree_id) ON DELETE CASCADE
);
