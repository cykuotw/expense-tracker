-- Preflight/remediation: if this migration aborts, inspect conflicting account IDs with
-- SELECT LOWER(BTRIM(email)) AS normalized_email, ARRAY_AGG(id ORDER BY create_time_utc)
-- FROM users GROUP BY LOWER(BTRIM(email)) HAVING COUNT(*) > 1;
-- Resolve each conflict according to account ownership, then rerun the migration.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        GROUP BY LOWER(BTRIM(email))
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot enforce normalized user email uniqueness: resolve duplicate normalized email rows first';
    END IF;
END $$;

UPDATE users SET email = LOWER(BTRIM(email));
UPDATE invitations SET email = LOWER(BTRIM(email));

CREATE UNIQUE INDEX users_email_normalized_unique_idx
ON users (LOWER(BTRIM(email)));
