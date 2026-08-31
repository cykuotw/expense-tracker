DROP INDEX IF EXISTS idx_invitations_registration_session_hash;

ALTER TABLE invitations
    DROP CONSTRAINT IF EXISTS invitations_registration_session_pair,
    DROP COLUMN IF EXISTS registration_session_expires_at,
    DROP COLUMN IF EXISTS registration_session_hash;

ALTER TABLE invitations RENAME COLUMN token_hash TO token;
