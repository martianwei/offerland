CREATE TABLE IF NOT EXISTS application_results (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    user_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    major_id uuid NOT NULL REFERENCES majors(major_id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    UNIQUE(user_id, major_id)
);