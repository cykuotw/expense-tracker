ALTER TABLE users
    ADD COLUMN is_protected_admin BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE users
    ADD CONSTRAINT users_protected_admin_requires_active_admin
    CHECK (
        NOT is_protected_admin
        OR (role IS NOT DISTINCT FROM 'admin' AND is_active IS TRUE)
    );

CREATE UNIQUE INDEX users_single_protected_admin_idx
    ON users (is_protected_admin)
    WHERE is_protected_admin IS TRUE;

CREATE OR REPLACE FUNCTION enforce_protected_admin_invariant()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' AND OLD.is_protected_admin IS TRUE THEN
        RAISE EXCEPTION 'protected administrator cannot be deleted'
            USING ERRCODE = '23514';
    END IF;

    IF TG_OP = 'UPDATE' AND OLD.is_protected_admin IS TRUE THEN
        IF NEW.is_protected_admin IS NOT TRUE
            OR NEW.role IS DISTINCT FROM 'admin'
            OR NEW.is_active IS DISTINCT FROM TRUE THEN
            RAISE EXCEPTION 'protected administrator invariant cannot be changed'
                USING ERRCODE = '23514';
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER users_protected_admin_invariant
    BEFORE UPDATE OR DELETE ON users
    FOR EACH ROW
    EXECUTE FUNCTION enforce_protected_admin_invariant();
