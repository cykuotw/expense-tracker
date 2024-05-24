CREATE TABLE IF NOT EXISTS expense_type(
    "id" UUID NOT NULL PRIMARY KEY,
    "category_id" UUID NOT NULL,
    "name" VARCHAR(32) NOT NULL
);

CREATE TABLE IF NOT EXISTS category(
    "id" UUID NOT NULL PRIMARY KEY,
    "name" VARCHAR(32) NOT NULL
);

-- insert default data