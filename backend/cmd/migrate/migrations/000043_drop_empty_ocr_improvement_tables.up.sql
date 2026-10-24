-- Contract release: deploy the Worker without historical evidence dependencies
-- before applying this migration. One DO statement keeps checks and DDL atomic.
DO $$
BEGIN
    -- Parent DDL locks first, matching expense -> receipt mutation order.
    -- Bootstrap bounds lock acquisition to five seconds.
    LOCK TABLE expense, expense_receipt,
        ocr_improvement_consent, ocr_improvement_evidence IN ACCESS EXCLUSIVE MODE;

    IF EXISTS (SELECT 1 FROM ocr_improvement_consent)
       OR EXISTS (SELECT 1 FROM ocr_improvement_evidence) THEN
        RAISE EXCEPTION 'historical OCR improvement tables are not empty; migration 000043 refused';
    END IF;

    DROP TRIGGER expense_evidence_cleanup ON expense;
    DROP TRIGGER receipt_evidence_cleanup ON expense_receipt;
    DROP FUNCTION mark_ocr_evidence_for_cleanup();
    DROP TABLE ocr_improvement_evidence;
    DROP TABLE ocr_improvement_consent;
END;
$$;
