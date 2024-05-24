CREATE TABLE IF NOT EXISTS item(
    "id" UUID NOT NULL PRIMARY KEY,
    "expense_id" UUID NOT NULL,
    "name" VARCHAR(32),
    "amount" NUMERIC(10,2),
    "unit" VARCHAR(10),
    "unit_price" NUMERIC(10,3)
);
