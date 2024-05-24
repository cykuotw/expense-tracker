CREATE TABLE IF NOT EXISTS expense(
    "id" UUID NOT NULL PRIMARY KEY,
    "description" VARCHAR(256) NOT NULL,
    "group_id" UUID NOT NULL,
    "create_by_user_id" UUID NOT NULL,
    "pay_by_user_id" UUID NOT NULL,
    "provider_name" VARCHAR(128),
    "exp_type_id" UUID NOT NULL,
    "is_settled" BOOLEAN NOT NULL,
    "sub_total" NUMERIC(10, 3),
    "tax_fee_tip" NUMERIC(10, 3),
    "total" NUMERIC(10, 3) NOT NULL,
    "currency" CHAR(3) NOT NULL,
    "invoice_pic_url" TEXT
);