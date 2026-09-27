ALTER TABLE item
    ALTER COLUMN name TYPE VARCHAR(256),
    ALTER COLUMN amount TYPE NUMERIC(15, 6),
    ALTER COLUMN unit TYPE VARCHAR(32),
    ADD COLUMN description VARCHAR(256),
    ADD COLUMN quantity NUMERIC(15, 6),
    ADD COLUMN confirmed_unit VARCHAR(32),
    ADD COLUMN confirmed_unit_price NUMERIC(10, 3),
    ADD COLUMN line_total NUMERIC(10, 3),
    ADD COLUMN position INTEGER;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM item
        WHERE name IS NULL
           OR BTRIM(name) = ''
           OR amount IS NULL
           OR amount <= 0
           OR unit_price IS NULL
           OR unit_price < 0
           OR ROUND(amount * unit_price, 3) > 9999999.999
    ) THEN
        RAISE EXCEPTION 'item rows cannot be unambiguously backfilled into the confirmed item contract';
    END IF;
END
$$;

UPDATE item
SET description = BTRIM(name),
    quantity = amount,
    confirmed_unit = NULLIF(BTRIM(unit), ''),
    confirmed_unit_price = unit_price,
    line_total = ROUND(amount * unit_price, 3);

WITH ordered_items AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY expense_id ORDER BY id) - 1 AS resolved_position
    FROM item
)
UPDATE item
SET position = ordered_items.resolved_position
FROM ordered_items
WHERE item.id = ordered_items.id;

ALTER TABLE item
    ADD CONSTRAINT item_position_nonnegative CHECK (position IS NULL OR position >= 0);

CREATE UNIQUE INDEX item_expense_position_unique
    ON item (expense_id, position)
    WHERE position IS NOT NULL;
