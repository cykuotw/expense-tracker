DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM expense AS e
        LEFT JOIN ledger AS l ON l.expense_id = e.id
        GROUP BY e.id, e.total
        HAVING COUNT(l.id) = 0 OR COALESCE(SUM(l.share), 0) <> e.total
    ) THEN
        RAISE EXCEPTION 'cannot migrate expenses with missing or inconsistent ledgers';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM ledger
        GROUP BY expense_id, borrower_user_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot migrate duplicate expense ledger participants';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM expense AS e
        JOIN ledger AS l ON l.expense_id = e.id
        LEFT JOIN group_member AS gm
            ON gm.group_id = e.group_id
            AND gm.user_id = l.borrower_user_id
        WHERE gm.user_id IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot migrate expense participants outside their group';
    END IF;
END $$;

ALTER TABLE ledger
    ADD CONSTRAINT ledger_expense_borrower_unique
    UNIQUE (expense_id, borrower_user_id);

ALTER TABLE expense
    ADD COLUMN allocation_mode TEXT
    CHECK (allocation_mode IN ('equal', 'exact', 'percentage', 'adjustment'));

CREATE TABLE expense_allocation (
    expense_id UUID NOT NULL,
    user_id UUID NOT NULL,
    amount NUMERIC(10, 3),
    percentage_basis_points INTEGER,
    PRIMARY KEY (expense_id, user_id),
    CONSTRAINT expense_allocation_expense_fk
        FOREIGN KEY (expense_id) REFERENCES expense(id) ON DELETE CASCADE,
    CONSTRAINT expense_allocation_user_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT expense_allocation_amount_non_negative
        CHECK (amount IS NULL OR amount >= 0),
    CONSTRAINT expense_allocation_percentage_bounds
        CHECK (
            percentage_basis_points IS NULL OR
            percentage_basis_points BETWEEN 0 AND 10000
        )
);

INSERT INTO expense_allocation (
    expense_id,
    user_id,
    amount,
    percentage_basis_points
)
SELECT
    e.id,
    l.borrower_user_id,
    CASE
        WHEN e.split_rule IN ('Unequally', 'You-Full', 'Other-Full')
            THEN l.share
        ELSE NULL
    END,
    NULL
FROM expense AS e
JOIN ledger AS l ON l.expense_id = e.id;

UPDATE expense
SET allocation_mode = CASE
    WHEN split_rule IN ('Equally', 'You-Half', 'Other-Half') THEN 'equal'
    ELSE 'exact'
END;

ALTER TABLE expense
    ALTER COLUMN allocation_mode SET NOT NULL;

ALTER TABLE expense
    DROP COLUMN split_rule;
