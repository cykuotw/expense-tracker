DELETE FROM web_push_delivery
WHERE notification_type = 'monthly_review_available';

DROP INDEX expense_monthly_review_page_idx;
DROP INDEX monthly_review_publication_published_at_idx;
DROP INDEX web_push_delivery_monthly_review_subscription_unique;
DROP INDEX web_push_delivery_expense_subscription_unique;

ALTER TABLE web_push_delivery
    DROP CONSTRAINT web_push_delivery_event_shape_check,
    DROP CONSTRAINT web_push_delivery_notification_type_check,
    DROP COLUMN review_month,
    DROP COLUMN notification_type,
    ALTER COLUMN expense_id SET NOT NULL,
    ALTER COLUMN actor_user_id SET NOT NULL,
    ALTER COLUMN currency SET NOT NULL,
    ALTER COLUMN amount SET NOT NULL,
    ADD CONSTRAINT web_push_delivery_expense_id_subscription_id_key
        UNIQUE (expense_id, subscription_id);

DROP TABLE monthly_review_publication;
