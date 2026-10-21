CREATE TABLE ocr_improvement_consent (
    account_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    policy_version TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    capture_bucket TEXT NOT NULL DEFAULT '',
    purge_pending BOOLEAN NOT NULL DEFAULT FALSE,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ocr_improvement_evidence (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES users(id),
    run_id UUID NOT NULL UNIQUE,
    expense_id UUID REFERENCES expense(id) ON DELETE RESTRICT,
    receipt_id UUID REFERENCES expense_receipt(id) ON DELETE SET NULL,
    consent_revision BIGINT NOT NULL CHECK (consent_revision > 0),
    policy_version TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending', 'confirmed', 'deleting', 'deleted')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ocr_evidence_account_state ON ocr_improvement_evidence(account_id, state, id);
CREATE INDEX ocr_evidence_expense ON ocr_improvement_evidence(expense_id) WHERE expense_id IS NOT NULL;

CREATE FUNCTION mark_ocr_evidence_for_cleanup() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF TG_TABLE_NAME = 'expense' THEN
        IF NEW.is_deleted AND NOT OLD.is_deleted THEN
            UPDATE ocr_improvement_evidence SET state='deleting', updated_at=NOW()
            WHERE expense_id=NEW.id AND state<>'deleted';
        END IF;
    ELSE
        IF NEW.status IN ('deleting','deleted') AND OLD.status NOT IN ('deleting','deleted') THEN
            UPDATE ocr_improvement_evidence SET state='deleting', updated_at=NOW()
            WHERE expense_id=NEW.expense_id AND state<>'deleted';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER expense_evidence_cleanup AFTER UPDATE OF is_deleted ON expense
FOR EACH ROW EXECUTE FUNCTION mark_ocr_evidence_for_cleanup();
CREATE TRIGGER receipt_evidence_cleanup AFTER UPDATE OF status ON expense_receipt
FOR EACH ROW EXECUTE FUNCTION mark_ocr_evidence_for_cleanup();
