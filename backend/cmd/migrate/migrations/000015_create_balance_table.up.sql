CREATE TABLE IF NOT EXISTS balance(
    "id" uuid NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    "sender_user_id" uuid NOT NULL,
    "receiver_user_id" uuid NOT NULL,
    "share" numeric(10, 3) NOT NULL,
    "group_id" uuid NOT NULL,
    "create_time_utc" timestamp NOT NULL DEFAULT now(),
    "is_outdated" boolean NOT NULL DEFAULT FALSE,
    "update_time_utc" timestamp,
    "is_settled" boolean NOT NULL DEFAULT FALSE,
    "settle_time_utc" timestamp
);

CREATE TABLE IF NOT EXISTS balance_ledger(
    "id" uuid NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    "balance_id" uuid NOT NULL,
    "ledger_id" uuid NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_balance_id ON balance_ledger(balance_id);

CREATE INDEX IF NOT EXISTS idx_ledger_id ON balance_ledger(ledger_id);

