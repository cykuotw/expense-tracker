CREATE TABLE user_feature_grant (
    user_id UUID NOT NULL,
    feature_key VARCHAR(64) NOT NULL,
    granted_by_user_id UUID NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, feature_key),
    CONSTRAINT user_feature_grant_user_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT user_feature_grant_actor_fk
        FOREIGN KEY (granted_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT user_feature_grant_supported_feature
        CHECK (feature_key IN ('receipt_ocr'))
);

CREATE INDEX user_feature_grant_actor_idx
    ON user_feature_grant(granted_by_user_id);
