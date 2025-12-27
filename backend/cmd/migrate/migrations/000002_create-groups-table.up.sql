CREATE TABLE IF NOT EXISTS groups(
    "id" UUID NOT NULL PRIMARY KEY,
    "group_name" VARCHAR(32) NOT NULL,
    "description" VARCHAR(256),
    "create_time_utc" TIMESTAMP WITH TIME ZONE NOT NULL,
    "is_active" BOOLEAN NOT NULL,
    "create_by_user_id" UUID NOT NULL
);