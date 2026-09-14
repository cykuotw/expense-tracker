CREATE TABLE monthly_review_publication (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    review_month DATE NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, review_month),
    CONSTRAINT monthly_review_publication_month_start_check
        CHECK (review_month = date_trunc('month', review_month)::date)
);

ALTER TABLE web_push_delivery
    DROP CONSTRAINT web_push_delivery_expense_id_subscription_id_key,
    ALTER COLUMN expense_id DROP NOT NULL,
    ALTER COLUMN actor_user_id DROP NOT NULL,
    ALTER COLUMN currency DROP NOT NULL,
    ALTER COLUMN amount DROP NOT NULL,
    ADD COLUMN notification_type VARCHAR(32) NOT NULL DEFAULT 'expense_created',
    ADD COLUMN review_month DATE,
    ADD CONSTRAINT web_push_delivery_notification_type_check
        CHECK (notification_type IN ('expense_created', 'monthly_review_available')),
    ADD CONSTRAINT web_push_delivery_event_shape_check CHECK (
        (
            notification_type = 'expense_created'
            AND expense_id IS NOT NULL
            AND actor_user_id IS NOT NULL
            AND currency IS NOT NULL
            AND amount IS NOT NULL
            AND review_month IS NULL
        ) OR (
            notification_type = 'monthly_review_available'
            AND expense_id IS NULL
            AND actor_user_id IS NULL
            AND currency IS NULL
            AND amount IS NULL
            AND review_month IS NOT NULL
            AND review_month = date_trunc('month', review_month)::date
        )
    );

CREATE UNIQUE INDEX web_push_delivery_expense_subscription_unique
    ON web_push_delivery (expense_id, subscription_id)
    WHERE notification_type = 'expense_created';

CREATE UNIQUE INDEX web_push_delivery_monthly_review_subscription_unique
    ON web_push_delivery (group_id, review_month, subscription_id)
    WHERE notification_type = 'monthly_review_available';

CREATE INDEX monthly_review_publication_published_at_idx
    ON monthly_review_publication (published_at, group_id, review_month);

CREATE INDEX expense_monthly_review_page_idx
    ON expense (
        group_id,
        currency,
        (COALESCE(occurred_on, (expense_time_utc AT TIME ZONE 'UTC')::date)) DESC,
        expense_time_utc DESC,
        id DESC
    )
    WHERE is_deleted IS FALSE;
