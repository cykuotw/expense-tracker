ALTER TABLE expense
    ADD COLUMN occurred_on date;

ALTER TABLE expense
    ALTER COLUMN delete_time_utc TYPE timestamp with time zone
        USING delete_time_utc AT TIME ZONE 'UTC',
    ALTER COLUMN settle_time_utc TYPE timestamp with time zone
        USING settle_time_utc AT TIME ZONE 'UTC';

ALTER TABLE balance
    ALTER COLUMN create_time_utc TYPE timestamp with time zone
        USING create_time_utc AT TIME ZONE 'UTC',
    ALTER COLUMN update_time_utc TYPE timestamp with time zone
        USING update_time_utc AT TIME ZONE 'UTC',
    ALTER COLUMN settle_time_utc TYPE timestamp with time zone
        USING settle_time_utc AT TIME ZONE 'UTC';

ALTER TABLE invitations
    ALTER COLUMN expires_at TYPE timestamp with time zone
        USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN used_at TYPE timestamp with time zone
        USING used_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE timestamp with time zone
        USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN registration_session_expires_at TYPE timestamp with time zone
        USING registration_session_expires_at AT TIME ZONE 'UTC';

ALTER TABLE refresh_tokens
    ALTER COLUMN expires_at TYPE timestamp with time zone
        USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN revoked_at TYPE timestamp with time zone
        USING revoked_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE timestamp with time zone
        USING created_at AT TIME ZONE 'UTC';
