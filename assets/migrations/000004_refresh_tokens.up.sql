CREATE TABLE IF NOT EXISTS refresh_tokens (
    token bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL,
    expires_at timestamp(0) with time zone NOT NULL
);