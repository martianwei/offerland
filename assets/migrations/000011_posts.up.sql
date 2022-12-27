CREATE TABLE IF NOT EXISTS posts (
    post_id uuid NOT NULL PRIMARY KEY,
    add_result BOOLEAN NOT NULL,
    body text NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    user_id varchar(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE
);