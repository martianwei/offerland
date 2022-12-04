CREATE TABLE user_to_results (
    user_id uuid NOT NULL,
    school_name VARCHAR(255) NOT NULL,
    major_name VARCHAR(255) NOT NULL,
    announce_date DATE NOT NULL,
    status VARCHAR(255) NOT NULL,
    others VARCHAR(255) NOT NULL,
    PRIMARY KEY (user_id, school_name, major_name, announce_date)
);