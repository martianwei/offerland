CREATE TABLE IF NOT EXISTS reset_tokens (
    hash bytea PRIMARY KEY,
    user_id varchar(255) NOT NULL REFERENCES users ON DELETE CASCADE,
    expiry timestamp(0) with time zone NOT NULL
);