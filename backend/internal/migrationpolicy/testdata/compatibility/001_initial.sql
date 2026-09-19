CREATE TABLE expense_contract_fixture (
    id UUID PRIMARY KEY,
    description TEXT NOT NULL,
    total NUMERIC(10, 3) NOT NULL,
    expense_time TIMESTAMP WITHOUT TIME ZONE NOT NULL
);
