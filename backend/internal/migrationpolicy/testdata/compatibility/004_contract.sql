ALTER TABLE expense_contract_fixture
    ALTER COLUMN occurred_on SET NOT NULL;

ALTER TABLE expense_contract_fixture
    DROP CONSTRAINT expense_contract_fixture_occurred_on_present;
