DROP TRIGGER receipt_evidence_cleanup ON expense_receipt;
DROP TRIGGER expense_evidence_cleanup ON expense;
DROP FUNCTION mark_ocr_evidence_for_cleanup();
DROP TABLE ocr_improvement_evidence;
DROP TABLE ocr_improvement_consent;
