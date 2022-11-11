CREATE TABLE IF NOT EXISTS activation_tokens (
    hash bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    passcode text NOT NULL,
    expiry timestamp(0) with time zone NOT NULL
);