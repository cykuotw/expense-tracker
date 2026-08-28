ALTER TABLE users
    ADD COLUMN has_local_password BOOLEAN;

UPDATE users
SET has_local_password = external_type IS NULL;

ALTER TABLE users
    ALTER COLUMN has_local_password SET NOT NULL,
    ALTER COLUMN has_local_password SET DEFAULT TRUE;

ALTER TABLE users
    ADD CONSTRAINT users_password_capability_consistent_chk
    CHECK (has_local_password OR (external_type IS NOT NULL AND external_id IS NOT NULL));
