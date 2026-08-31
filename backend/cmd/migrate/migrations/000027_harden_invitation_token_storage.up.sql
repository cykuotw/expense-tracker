CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE invitations RENAME COLUMN token TO token_hash;

UPDATE invitations
SET token_hash = encode(digest(token_hash, 'sha256'), 'hex'),
    expires_at = CASE
        WHEN used_at IS NULL AND expires_at > NOW() THEN NOW()
        ELSE expires_at
    END;

ALTER TABLE invitations
    ADD COLUMN registration_session_hash varchar(255),
    ADD COLUMN registration_session_expires_at timestamp;

ALTER TABLE invitations
    ADD CONSTRAINT invitations_registration_session_pair
    CHECK (
        (registration_session_hash IS NULL AND registration_session_expires_at IS NULL)
        OR (registration_session_hash IS NOT NULL AND registration_session_expires_at IS NOT NULL)
    );

CREATE UNIQUE INDEX idx_invitations_registration_session_hash
    ON invitations (registration_session_hash)
    WHERE registration_session_hash IS NOT NULL;
