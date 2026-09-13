-- Prevent writes while the preflight and constraint installation run so a
-- duplicate cannot be inserted between the two steps.
LOCK TABLE balance_ledger IN SHARE ROW EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM balance_ledger
        GROUP BY balance_id, ledger_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            CONSTRAINT = 'balance_ledger_balance_id_ledger_id_unique',
            MESSAGE = 'cannot enforce balance-ledger uniqueness: resolve duplicate (balance_id, ledger_id) rows first';
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'balance_ledger'::regclass
          AND conname = 'balance_ledger_balance_id_ledger_id_unique'
    ) THEN
        ALTER TABLE balance_ledger
            ADD CONSTRAINT balance_ledger_balance_id_ledger_id_unique
            UNIQUE (balance_id, ledger_id);
    END IF;
END
$$;
