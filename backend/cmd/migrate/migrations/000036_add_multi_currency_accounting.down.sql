DROP TRIGGER IF EXISTS balance_ledger_currency_before_write ON balance_ledger;
DROP FUNCTION IF EXISTS populate_balance_ledger_currency();
ALTER TABLE balance_ledger
    DROP CONSTRAINT IF EXISTS balance_ledger_ledger_currency_fk,
    DROP CONSTRAINT IF EXISTS balance_ledger_balance_currency_fk,
    DROP COLUMN IF EXISTS currency;

DROP TRIGGER IF EXISTS balance_currency_before_write ON balance;
DROP FUNCTION IF EXISTS populate_balance_currency();
DROP INDEX IF EXISTS idx_balance_group_currency_open;
ALTER TABLE balance
    DROP CONSTRAINT IF EXISTS balance_id_currency_unique,
    DROP CONSTRAINT IF EXISTS balance_currency_fk,
    DROP COLUMN IF EXISTS currency;

DROP TRIGGER IF EXISTS ledger_currency_before_write ON ledger;
DROP FUNCTION IF EXISTS populate_ledger_currency();
ALTER TABLE ledger
    DROP CONSTRAINT IF EXISTS ledger_id_currency_unique,
    DROP CONSTRAINT IF EXISTS ledger_currency_fk,
    DROP COLUMN IF EXISTS currency;

ALTER TABLE expense DROP CONSTRAINT IF EXISTS expense_group_currency_fk;
DROP INDEX IF EXISTS idx_expense_group_currency;

DROP TRIGGER IF EXISTS groups_currency_compatibility_after_write ON groups;
DROP FUNCTION IF EXISTS ensure_group_currency_compatibility();
DROP TRIGGER IF EXISTS groups_currency_compatibility_before_write ON groups;
DROP FUNCTION IF EXISTS sync_group_currency_compatibility();

ALTER TABLE groups
    DROP CONSTRAINT IF EXISTS groups_settlement_preview_group_currency_fk,
    DROP CONSTRAINT IF EXISTS groups_settlement_preview_currency_fk,
    DROP COLUMN IF EXISTS settlement_preview_currency;

DROP TABLE IF EXISTS group_currency;
