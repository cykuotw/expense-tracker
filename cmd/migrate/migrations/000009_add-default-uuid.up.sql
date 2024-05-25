ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE expense ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE expense_type ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE group_member ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE groups ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE item ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE ledger ALTER COLUMN id SET DEFAULT gen_random_uuid();