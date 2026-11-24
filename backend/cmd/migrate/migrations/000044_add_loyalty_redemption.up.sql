ALTER TABLE expense
    ADD COLUMN loyalty_redemption NUMERIC,
    ADD CONSTRAINT expense_loyalty_redemption_amounts CHECK (
        loyalty_redemption IS NULL OR (
            loyalty_redemption >= 0 AND total >= 0
            AND sub_total + tax_fee_tip > 0
            AND total = sub_total + tax_fee_tip - loyalty_redemption
        )
    ) NOT VALID;
