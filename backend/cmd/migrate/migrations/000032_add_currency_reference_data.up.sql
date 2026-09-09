-- ISO 4217 List One: https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/lists/list-one.xml
-- Source publication date: 2026-01-01. Retrieved: 2026-09-08.
CREATE TABLE currency (
    code CHAR(3) PRIMARY KEY,
    display_name TEXT NOT NULL,
    minor_unit_digits SMALLINT NOT NULL,
    amount_digits SMALLINT NOT NULL,
    CHECK (code ~ '^[A-Z]{3}$'),
    CHECK (btrim(display_name) <> ''),
    CHECK (minor_unit_digits BETWEEN 0 AND 3),
    CHECK (amount_digits BETWEEN 0 AND minor_unit_digits)
);

INSERT INTO currency (code, display_name, minor_unit_digits, amount_digits) VALUES
    ('CAD', 'Canadian Dollar', 2, 2),
    ('USD', 'US Dollar', 2, 2),
    ('TWD', 'New Taiwan Dollar', 2, 0),
    ('EUR', 'Euro', 2, 2),
    ('GBP', 'British Pound', 2, 2),
    ('JPY', 'Japanese Yen', 0, 0),
    ('CNY', 'Chinese Yuan', 2, 2),
    ('HKD', 'Hong Kong Dollar', 2, 2),
    ('KRW', 'South Korean Won', 0, 0),
    ('SGD', 'Singapore Dollar', 2, 2),
    ('AUD', 'Australian Dollar', 2, 2),
    ('NZD', 'New Zealand Dollar', 2, 2),
    ('CHF', 'Swiss Franc', 2, 2),
    ('THB', 'Thai Baht', 2, 2),
    ('MYR', 'Malaysian Ringgit', 2, 2),
    ('PHP', 'Philippine Peso', 2, 2),
    ('IDR', 'Indonesian Rupiah', 2, 2),
    ('VND', 'Vietnamese Dong', 0, 0),
    ('INR', 'Indian Rupee', 2, 2),
    ('MXN', 'Mexican Peso', 2, 2);

-- Earlier releases used the non-ISO code NTD for New Taiwan Dollar.
-- Convert both snapshots before validating the supported ISO set so existing
-- groups and their expenses retain the same currency meaning.
UPDATE groups SET currency = 'TWD' WHERE currency = 'NTD';
UPDATE expense SET currency = 'TWD' WHERE currency = 'NTD';

-- The old application allowed a group's selected currency to drift from its
-- expenses. A group with one historical expense currency can be repaired
-- without changing a transaction snapshot. Mixed-currency histories remain a
-- deliberate migration failure below and require an explicit data decision.
UPDATE groups AS groups
SET currency = expense_currency.currency
FROM (
    SELECT group_id, min(currency) AS currency
    FROM expense
    GROUP BY group_id
    HAVING count(DISTINCT currency) = 1
) AS expense_currency
WHERE groups.id = expense_currency.group_id
  AND groups.currency IS DISTINCT FROM expense_currency.currency;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM groups
        WHERE currency IS NOT NULL
          AND currency NOT IN (SELECT code FROM currency)
    ) THEN
        RAISE EXCEPTION 'unsupported existing group currency; migration aborted';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM expense
        WHERE currency NOT IN (SELECT code FROM currency)
    ) THEN
        RAISE EXCEPTION 'unsupported existing expense currency; migration aborted';
    END IF;
END $$;

UPDATE groups SET currency = 'CAD' WHERE currency IS NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM expense
        JOIN groups ON groups.id = expense.group_id
        WHERE expense.currency <> groups.currency
    ) THEN
        RAISE EXCEPTION 'existing expense currency does not match its group currency; migration aborted';
    END IF;
END $$;

ALTER TABLE groups
    ALTER COLUMN currency SET NOT NULL,
    ADD CONSTRAINT groups_currency_fk FOREIGN KEY (currency) REFERENCES currency(code) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE expense
    ADD CONSTRAINT expense_currency_fk FOREIGN KEY (currency) REFERENCES currency(code) ON UPDATE RESTRICT ON DELETE RESTRICT;
