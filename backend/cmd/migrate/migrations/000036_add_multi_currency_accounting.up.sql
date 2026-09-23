ALTER TABLE groups
    ADD COLUMN settlement_preview_currency CHAR(3);

CREATE TABLE group_currency (
    group_id UUID NOT NULL,
    currency CHAR(3) NOT NULL,
    enabled_for_new_expenses BOOLEAN NOT NULL DEFAULT FALSE,
    preview_rate NUMERIC(30, 15),
    PRIMARY KEY (group_id, currency),
    CONSTRAINT group_currency_group_fk
        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT group_currency_currency_fk
        FOREIGN KEY (currency) REFERENCES currency(code) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT group_currency_preview_rate_positive
        CHECK (preview_rate IS NULL OR preview_rate > 0)
);

INSERT INTO group_currency (group_id, currency, enabled_for_new_expenses, preview_rate)
SELECT id, currency, TRUE, 1
FROM groups;

INSERT INTO group_currency (group_id, currency, enabled_for_new_expenses, preview_rate)
SELECT DISTINCT expense.group_id, expense.currency, FALSE, NULL::NUMERIC(30, 15)
FROM expense
ON CONFLICT (group_id, currency) DO NOTHING;

UPDATE groups
SET settlement_preview_currency = currency
WHERE settlement_preview_currency IS NULL;

ALTER TABLE groups
    ALTER COLUMN settlement_preview_currency SET NOT NULL,
    ADD CONSTRAINT groups_settlement_preview_currency_fk
        FOREIGN KEY (settlement_preview_currency)
        REFERENCES currency(code) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT groups_settlement_preview_group_currency_fk
        FOREIGN KEY (id, settlement_preview_currency)
        REFERENCES group_currency(group_id, currency)
        DEFERRABLE INITIALLY DEFERRED;

CREATE FUNCTION sync_group_currency_compatibility() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.settlement_preview_currency IS NULL THEN
        NEW.settlement_preview_currency := NEW.currency;
    ELSIF TG_OP = 'UPDATE'
        AND NEW.currency IS DISTINCT FROM OLD.currency
        AND NEW.settlement_preview_currency IS NOT DISTINCT FROM OLD.settlement_preview_currency
        AND OLD.settlement_preview_currency IS NOT DISTINCT FROM OLD.currency THEN
        NEW.settlement_preview_currency := NEW.currency;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER groups_currency_compatibility_before_write
BEFORE INSERT OR UPDATE OF currency, settlement_preview_currency ON groups
FOR EACH ROW EXECUTE FUNCTION sync_group_currency_compatibility();

CREATE FUNCTION ensure_group_currency_compatibility() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO group_currency (group_id, currency, enabled_for_new_expenses, preview_rate)
    VALUES (
        NEW.id,
        NEW.currency,
        TRUE,
        CASE WHEN NEW.currency = NEW.settlement_preview_currency THEN 1 ELSE NULL END
    )
    ON CONFLICT (group_id, currency) DO UPDATE
    SET enabled_for_new_expenses = TRUE,
        preview_rate = CASE
            WHEN EXCLUDED.currency = NEW.settlement_preview_currency THEN 1
            ELSE group_currency.preview_rate
        END;

    INSERT INTO group_currency (group_id, currency, enabled_for_new_expenses, preview_rate)
    VALUES (NEW.id, NEW.settlement_preview_currency, TRUE, 1)
    ON CONFLICT (group_id, currency) DO UPDATE
    SET preview_rate = 1;
    RETURN NEW;
END;
$$;

CREATE TRIGGER groups_currency_compatibility_after_write
AFTER INSERT OR UPDATE OF currency, settlement_preview_currency ON groups
FOR EACH ROW EXECUTE FUNCTION ensure_group_currency_compatibility();

CREATE INDEX idx_expense_group_currency ON expense(group_id, currency);

ALTER TABLE expense
    ADD CONSTRAINT expense_group_currency_fk
        FOREIGN KEY (group_id, currency)
        REFERENCES group_currency(group_id, currency)
        ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ledger ADD COLUMN currency CHAR(3);

UPDATE ledger
SET currency = expense.currency
FROM expense
WHERE ledger.expense_id = expense.id
  AND ledger.currency IS NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM ledger
        JOIN expense ON expense.id = ledger.expense_id
        WHERE ledger.currency IS DISTINCT FROM expense.currency
    ) THEN
        RAISE EXCEPTION 'ledger currency conflicts with expense currency; migration aborted';
    END IF;
END;
$$;

ALTER TABLE ledger
    ALTER COLUMN currency SET NOT NULL,
    ADD CONSTRAINT ledger_currency_fk
        FOREIGN KEY (currency) REFERENCES currency(code) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT ledger_id_currency_unique UNIQUE (id, currency);

CREATE FUNCTION populate_ledger_currency() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    expense_currency CHAR(3);
BEGIN
    SELECT currency INTO STRICT expense_currency
    FROM expense
    WHERE id = NEW.expense_id;

    IF NEW.currency IS NULL THEN
        NEW.currency := expense_currency;
    ELSIF NEW.currency IS DISTINCT FROM expense_currency THEN
        RAISE EXCEPTION 'ledger currency must match expense currency';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER ledger_currency_before_write
BEFORE INSERT OR UPDATE OF expense_id, currency ON ledger
FOR EACH ROW EXECUTE FUNCTION populate_ledger_currency();

ALTER TABLE balance ADD COLUMN currency CHAR(3);

UPDATE balance
SET currency = groups.currency
FROM groups
WHERE balance.group_id = groups.id
  AND balance.currency IS NULL;

ALTER TABLE balance
    ALTER COLUMN currency SET NOT NULL,
    ADD CONSTRAINT balance_currency_fk
        FOREIGN KEY (currency) REFERENCES currency(code) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT balance_id_currency_unique UNIQUE (id, currency);

CREATE INDEX idx_balance_group_currency_open
    ON balance(group_id, currency)
    WHERE is_outdated = FALSE AND is_settled = FALSE;

CREATE FUNCTION populate_balance_currency() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.currency IS NULL THEN
        SELECT currency INTO STRICT NEW.currency
        FROM groups
        WHERE id = NEW.group_id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER balance_currency_before_write
BEFORE INSERT OR UPDATE OF group_id, currency ON balance
FOR EACH ROW EXECUTE FUNCTION populate_balance_currency();

ALTER TABLE balance_ledger ADD COLUMN currency CHAR(3);

UPDATE balance_ledger
SET currency = ledger.currency
FROM ledger
WHERE balance_ledger.ledger_id = ledger.id
  AND balance_ledger.currency IS NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM balance_ledger
        JOIN balance ON balance.id = balance_ledger.balance_id
        JOIN ledger ON ledger.id = balance_ledger.ledger_id
        WHERE balance.currency IS DISTINCT FROM ledger.currency
    ) THEN
        RAISE EXCEPTION 'balance and ledger currency conflict; migration aborted';
    END IF;
END;
$$;

ALTER TABLE balance_ledger
    ALTER COLUMN currency SET NOT NULL,
    ADD CONSTRAINT balance_ledger_balance_currency_fk
        FOREIGN KEY (balance_id, currency)
        REFERENCES balance(id, currency) ON DELETE CASCADE,
    ADD CONSTRAINT balance_ledger_ledger_currency_fk
        FOREIGN KEY (ledger_id, currency)
        REFERENCES ledger(id, currency) ON DELETE CASCADE;

CREATE FUNCTION populate_balance_ledger_currency() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    ledger_currency CHAR(3);
    balance_currency CHAR(3);
BEGIN
    SELECT currency INTO STRICT ledger_currency FROM ledger WHERE id = NEW.ledger_id;
    SELECT currency INTO STRICT balance_currency FROM balance WHERE id = NEW.balance_id;
    IF ledger_currency IS DISTINCT FROM balance_currency THEN
        RAISE EXCEPTION 'balance and ledger currency must match';
    END IF;
    NEW.currency := ledger_currency;
    RETURN NEW;
END;
$$;

CREATE TRIGGER balance_ledger_currency_before_write
BEFORE INSERT OR UPDATE OF balance_id, ledger_id, currency ON balance_ledger
FOR EACH ROW EXECUTE FUNCTION populate_balance_ledger_currency();
