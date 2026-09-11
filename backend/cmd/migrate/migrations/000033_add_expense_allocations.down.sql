ALTER TABLE expense
    ADD COLUMN split_rule TEXT NOT NULL DEFAULT 'Equally'
    CHECK (split_rule IN ('Equally', 'Unequally', 'You-Half', 'You-Full', 'Other-Half', 'Other-Full'));

UPDATE expense
SET split_rule = CASE
    WHEN allocation_mode = 'equal' THEN 'Equally'
    ELSE 'Unequally'
END;

ALTER TABLE ledger
    DROP CONSTRAINT ledger_expense_borrower_unique;

DROP TABLE expense_allocation;

ALTER TABLE expense
    DROP COLUMN allocation_mode;
