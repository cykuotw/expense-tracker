-- Retired schema is not automatically recreated. Keep a compatible Worker or
-- design a reviewed forward recovery migration before restoring old consumers.
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000043 requires manual schema recovery; automatic reversal is unsupported';
END;
$$;
