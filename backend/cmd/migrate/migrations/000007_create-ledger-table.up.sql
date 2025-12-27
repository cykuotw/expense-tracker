CREATE TABLE IF NOT EXISTS ledger(
    "id" UUID NOT NULL PRIMARY KEY,
    "expense_id" UUID NOT NULL,
    "lender_user_id" UUID NOT NULL,
    "borrower_user_id" UUID NOT NULL,
    "share" NUMERIC(10, 3) NOT NULL
);