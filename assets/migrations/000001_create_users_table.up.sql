CREATE TYPE iss_type AS ENUM('google');
CREATE TABLE IF NOT EXISTS users (
    user_id uuid PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    username varchar(64) NOT NULL,
    email varchar(255) UNIQUE NOT NULL,
    password varchar(64) DEFAULT NULL,
    iss iss_type,
    sub varchar(255),
    activated bool NOT NULL,
    version integer NOT NULL DEFAULT 1
);