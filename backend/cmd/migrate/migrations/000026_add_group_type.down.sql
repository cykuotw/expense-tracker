ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_group_type_check;
ALTER TABLE groups DROP COLUMN IF EXISTS group_type;
