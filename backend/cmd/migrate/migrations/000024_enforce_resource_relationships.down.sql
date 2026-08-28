DROP INDEX IF EXISTS idx_balance_group_open;
DROP INDEX IF EXISTS idx_ledger_expense_id;
DROP INDEX IF EXISTS idx_item_expense_id;
DROP INDEX IF EXISTS idx_expense_group_id;

ALTER TABLE balance_ledger
    DROP CONSTRAINT IF EXISTS balance_ledger_ledger_fk,
    DROP CONSTRAINT IF EXISTS balance_ledger_balance_fk;

ALTER TABLE balance
    DROP CONSTRAINT IF EXISTS balance_receiver_fk,
    DROP CONSTRAINT IF EXISTS balance_sender_fk,
    DROP CONSTRAINT IF EXISTS balance_group_fk;

ALTER TABLE ledger
    DROP CONSTRAINT IF EXISTS ledger_borrower_fk,
    DROP CONSTRAINT IF EXISTS ledger_lender_fk,
    DROP CONSTRAINT IF EXISTS ledger_expense_fk;

ALTER TABLE item
    DROP CONSTRAINT IF EXISTS item_expense_fk;

ALTER TABLE expense
    DROP CONSTRAINT IF EXISTS expense_type_fk,
    DROP CONSTRAINT IF EXISTS expense_payer_fk,
    DROP CONSTRAINT IF EXISTS expense_creator_fk,
    DROP CONSTRAINT IF EXISTS expense_group_fk;

ALTER TABLE group_member
    DROP CONSTRAINT IF EXISTS group_member_group_user_unique,
    DROP CONSTRAINT IF EXISTS group_member_user_fk,
    DROP CONSTRAINT IF EXISTS group_member_group_fk;

ALTER TABLE groups
    DROP CONSTRAINT IF EXISTS groups_creator_fk;
