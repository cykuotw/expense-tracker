ALTER TABLE expense DROP CONSTRAINT IF EXISTS expense_currency_fk;

ALTER TABLE groups
    DROP CONSTRAINT IF EXISTS groups_currency_fk,
    ALTER COLUMN currency DROP NOT NULL;

DROP TABLE IF EXISTS currency;
