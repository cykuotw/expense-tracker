-- Security impact: rolling this migration back removes the database-level
-- protection that prevents the bootstrap administrator from being changed or deleted.
DROP TRIGGER IF EXISTS users_protected_admin_invariant ON users;
DROP FUNCTION IF EXISTS enforce_protected_admin_invariant();
DROP INDEX IF EXISTS users_single_protected_admin_idx;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_protected_admin_requires_active_admin;
ALTER TABLE users DROP COLUMN IF EXISTS is_protected_admin;
