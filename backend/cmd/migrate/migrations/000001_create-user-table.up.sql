CREATE TABLE IF NOT EXISTS users(
    "id" UUID NOT NULL PRIMARY KEY,
    "username" VARCHAR(32) NOT NULL,
    "firstname" VARCHAR(32) NOT NULL,
    "lastname" VARCHAR(32) NOT NULL,
    "email" VARCHAR(256) NOT NULL,
    "password_hash" VARCHAR(256) NOT NULL,
    "external_type" VARCHAR(16),
    "external_id" VARCHAR(64),
    "create_time_utc" TIMESTAMP WITH TIME ZONE NOT NULL,
    "is_active" BOOLEAN NOT NULL
);