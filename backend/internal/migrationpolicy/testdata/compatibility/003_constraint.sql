ALTER TABLE expense_contract_fixture
    ADD CONSTRAINT expense_contract_fixture_occurred_on_present
    CHECK (occurred_on IS NOT NULL) NOT VALID;

ALTER TABLE expense_contract_fixture
    VALIDATE CONSTRAINT expense_contract_fixture_occurred_on_present;
