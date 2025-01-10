ALTER TABLE expense
    ADD COLUMN delete_time_utc timestamp,
    ADD COLUMN settle_time_utc timestamp;

