UPDATE users
SET
    external_type = NULLIF(BTRIM(external_type), ''),
    external_id = NULLIF(BTRIM(external_id), '');

ALTER TABLE users
ADD CONSTRAINT users_external_identity_pair_consistent_chk CHECK (
    (external_type IS NULL AND external_id IS NULL)
    OR
    (
        external_type IS NOT NULL
        AND external_id IS NOT NULL
        AND BTRIM(external_type) <> ''
        AND BTRIM(external_id) <> ''
    )
);

CREATE UNIQUE INDEX users_external_identity_unique_idx
ON users (external_type, external_id)
WHERE external_type IS NOT NULL AND external_id IS NOT NULL;
