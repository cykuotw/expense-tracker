-- Linked local accounts lose their explicit password capability when this
-- migration is rolled back and will look Google-managed to older application code.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_password_capability_consistent_chk;
ALTER TABLE users DROP COLUMN IF EXISTS has_local_password;
