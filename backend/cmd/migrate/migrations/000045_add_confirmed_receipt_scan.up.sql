ALTER TABLE expense
    ADD COLUMN receipt_scan_note TEXT,
    ADD COLUMN ocr_request_id UUID;
