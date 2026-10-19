CREATE TABLE expense_category_breakdown (
    expense_id UUID NOT NULL REFERENCES expense(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position BETWEEN 0 AND 19),
    expense_type_id UUID NOT NULL REFERENCES expense_type(id) ON DELETE RESTRICT,
    amount NUMERIC NOT NULL CHECK (amount > 0 AND amount < 10000000),
    note TEXT NOT NULL DEFAULT '' CHECK (char_length(note) <= 200),
    PRIMARY KEY (expense_id, position),
    UNIQUE (expense_id, expense_type_id)
);

-- Effective classification includes legacy expenses without a data backfill.
CREATE VIEW expense_effective_category AS
SELECT expense_id, expense_type_id, amount FROM expense_category_breakdown
UNION ALL
SELECT e.id, e.exp_type_id, e.total FROM expense e
WHERE NOT EXISTS (SELECT 1 FROM expense_category_breakdown b WHERE b.expense_id = e.id);
