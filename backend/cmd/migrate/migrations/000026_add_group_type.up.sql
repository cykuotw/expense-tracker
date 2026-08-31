ALTER TABLE groups
    ADD COLUMN group_type VARCHAR(16) NOT NULL DEFAULT 'home',
    ADD CONSTRAINT groups_group_type_check
        CHECK (group_type IN ('trip', 'home', 'family', 'friends', 'event', 'other'));
