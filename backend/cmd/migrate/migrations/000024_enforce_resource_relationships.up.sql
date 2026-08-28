ALTER TABLE groups
    ADD CONSTRAINT groups_creator_fk
    FOREIGN KEY (create_by_user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE group_member
    ADD CONSTRAINT group_member_group_fk
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    ADD CONSTRAINT group_member_user_fk
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT group_member_group_user_unique UNIQUE (group_id, user_id);

ALTER TABLE expense
    ADD CONSTRAINT expense_group_fk
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE RESTRICT,
    ADD CONSTRAINT expense_creator_fk
    FOREIGN KEY (create_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT expense_payer_fk
    FOREIGN KEY (pay_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT expense_type_fk
    FOREIGN KEY (exp_type_id) REFERENCES expense_type(id) ON DELETE RESTRICT;

ALTER TABLE item
    ADD CONSTRAINT item_expense_fk
    FOREIGN KEY (expense_id) REFERENCES expense(id) ON DELETE CASCADE;

ALTER TABLE ledger
    ADD CONSTRAINT ledger_expense_fk
    FOREIGN KEY (expense_id) REFERENCES expense(id) ON DELETE CASCADE,
    ADD CONSTRAINT ledger_lender_fk
    FOREIGN KEY (lender_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT ledger_borrower_fk
    FOREIGN KEY (borrower_user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE balance
    ADD CONSTRAINT balance_group_fk
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    ADD CONSTRAINT balance_sender_fk
    FOREIGN KEY (sender_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT balance_receiver_fk
    FOREIGN KEY (receiver_user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE balance_ledger
    ADD CONSTRAINT balance_ledger_balance_fk
    FOREIGN KEY (balance_id) REFERENCES balance(id) ON DELETE CASCADE,
    ADD CONSTRAINT balance_ledger_ledger_fk
    FOREIGN KEY (ledger_id) REFERENCES ledger(id) ON DELETE CASCADE;

CREATE INDEX idx_expense_group_id ON expense(group_id);
CREATE INDEX idx_item_expense_id ON item(expense_id);
CREATE INDEX idx_ledger_expense_id ON ledger(expense_id);
CREATE INDEX idx_balance_group_open ON balance(group_id) WHERE is_outdated = FALSE AND is_settled = FALSE;
