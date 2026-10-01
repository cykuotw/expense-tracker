CREATE TABLE expense_receipt (
    id UUID PRIMARY KEY,
    expense_id UUID NOT NULL REFERENCES expense(id) ON DELETE RESTRICT,
    account_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    object_id UUID NOT NULL UNIQUE,
    temporary_key TEXT NOT NULL UNIQUE,
    retained_key TEXT NOT NULL UNIQUE,
    request_key UUID NOT NULL,
    token_sha256 CHAR(64) NOT NULL,
    delete_request_key UUID,
    role VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL,
    content_type VARCHAR(64) NOT NULL,
    byte_size BIGINT NOT NULL,
    checksum_sha256 CHAR(64) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    source_expires_at TIMESTAMPTZ NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error_code VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finalized_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT expense_receipt_role_check CHECK (role IN ('current', 'candidate', 'cleanup')),
    CONSTRAINT expense_receipt_status_check CHECK (status IN ('pending', 'finalized', 'failed', 'deleting', 'deleted')),
    CONSTRAINT expense_receipt_size_check CHECK (byte_size > 0 AND width > 0 AND height > 0),
    CONSTRAINT expense_receipt_request_unique UNIQUE (expense_id, request_key)
);
CREATE UNIQUE INDEX expense_receipt_one_current_idx ON expense_receipt (expense_id) WHERE role = 'current';
CREATE UNIQUE INDEX expense_receipt_one_candidate_idx ON expense_receipt (expense_id) WHERE role = 'candidate';
CREATE INDEX expense_receipt_reconcile_idx ON expense_receipt (updated_at, id) WHERE status IN ('pending', 'deleting');
CREATE UNIQUE INDEX expense_receipt_delete_request_idx ON expense_receipt (expense_id, delete_request_key) WHERE delete_request_key IS NOT NULL;
