DROP INDEX IF EXISTS item_expense_position_unique;

ALTER TABLE item
    DROP CONSTRAINT IF EXISTS item_position_nonnegative,
    DROP COLUMN IF EXISTS position,
    DROP COLUMN IF EXISTS line_total,
    DROP COLUMN IF EXISTS confirmed_unit_price,
    DROP COLUMN IF EXISTS confirmed_unit,
    DROP COLUMN IF EXISTS quantity,
    DROP COLUMN IF EXISTS description;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM item
        WHERE LENGTH(name) > 32
           OR LENGTH(unit) > 10
           OR amount <> ROUND(amount, 2)
           OR ABS(amount) > 99999999.99
    ) THEN
        RAISE EXCEPTION 'item legacy columns cannot be safely narrowed';
    END IF;
END
$$;

ALTER TABLE item
    ALTER COLUMN name TYPE VARCHAR(32),
    ALTER COLUMN amount TYPE NUMERIC(10, 2),
    ALTER COLUMN unit TYPE VARCHAR(10);
