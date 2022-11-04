CREATE TABLE IF NOT EXISTS jwt_tokens (
    token bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    expiry timestamp(0) with time zone NOT NULL
);