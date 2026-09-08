ALTER TABLE web_push_delivery
    ADD COLUMN claim_token UUID,
    ADD COLUMN claimed_until TIMESTAMPTZ;
