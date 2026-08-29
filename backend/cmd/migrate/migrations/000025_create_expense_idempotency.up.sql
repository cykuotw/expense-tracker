CREATE TABLE expense_create_idempotency (
    creator_user_id UUID NOT NULL,
    idempotency_key UUID NOT NULL,
    request_fingerprint BYTEA NOT NULL,
    expense_id UUID NOT NULL,
    created_at_utc TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (creator_user_id, idempotency_key),
    CONSTRAINT expense_create_idempotency_creator_fk
        FOREIGN KEY (creator_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT expense_create_idempotency_expense_fk
        FOREIGN KEY (expense_id) REFERENCES expense(id) ON DELETE RESTRICT
        DEFERRABLE INITIALLY DEFERRED
);
