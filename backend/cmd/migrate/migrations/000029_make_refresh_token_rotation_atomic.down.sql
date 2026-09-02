DROP INDEX idx_refresh_tokens_family_id;

ALTER TABLE refresh_tokens
    DROP COLUMN family_id;
