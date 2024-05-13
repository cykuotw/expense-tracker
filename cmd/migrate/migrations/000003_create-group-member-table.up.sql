CREATE TABLE IF NOT EXISTS group_member(
    "id" UUID NOT NULL PRIMARY KEY,
    "group_id" UUID NOT NULL,
    "user_id" UUID NOT NULL
);