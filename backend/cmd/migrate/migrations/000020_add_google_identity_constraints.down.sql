DROP INDEX IF EXISTS users_external_identity_unique_idx;

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_external_identity_pair_consistent_chk;
