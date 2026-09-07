package notification

import (
	"context"
	"database/sql"
	"errors"
	"expense-tracker/backend/types"
	"time"

	"github.com/google/uuid"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) UpsertSubscription(ctx context.Context, userID uuid.UUID, input types.WebPushSubscriptionInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", userID.String()); err != nil {
		return err
	}
	var existingOwner uuid.UUID
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM web_push_subscription WHERE endpoint = $1", input.Endpoint).Scan(&existingOwner)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && existingOwner != userID {
		return types.ErrPermissionDenied
	}
	if errors.Is(err, sql.ErrNoRows) {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM web_push_subscription WHERE user_id = $1", userID).Scan(&count); err != nil {
			return err
		}
		if count >= types.WebPushSubscriptionLimit {
			return types.ErrWebPushSubscriptionLimit
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO web_push_subscription (user_id, endpoint, p256dh_key, auth_key, show_details)
		VALUES ($1, $2, $3, $4, FALSE)
		ON CONFLICT (endpoint) DO UPDATE
		SET p256dh_key = EXCLUDED.p256dh_key,
			auth_key = EXCLUDED.auth_key,
			show_details = FALSE,
			updated_at = NOW()
		WHERE web_push_subscription.user_id = EXCLUDED.user_id`,
		userID, input.Endpoint, input.P256DH, input.Auth,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM web_push_subscription WHERE user_id = $1 AND endpoint = $2", userID, endpoint)
	return err
}

func (s *Store) SetDetails(ctx context.Context, userID uuid.UUID, endpoint string, showDetails bool) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE web_push_subscription
		SET show_details = $3, updated_at = NOW()
		WHERE user_id = $1 AND endpoint = $2`, userID, endpoint, showDetails)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return types.ErrPermissionDenied
	}
	return nil
}

func (s *Store) SetGroupMute(ctx context.Context, userID, groupID uuid.UUID, muted bool) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO group_notification_preference (group_id, user_id, muted)
		SELECT $1, $2, $3
		WHERE EXISTS (
			SELECT 1 FROM group_member WHERE group_id = $1 AND user_id = $2
		)
		ON CONFLICT (group_id, user_id) DO UPDATE
		SET muted = EXCLUDED.muted, updated_at = NOW()`, groupID, userID, muted)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return types.ErrPermissionDenied
	}
	return nil
}

func (s *Store) Settings(ctx context.Context, userID uuid.UUID, publicKey string) (types.WebPushSettings, error) {
	settings := types.WebPushSettings{VAPIDKey: publicKey, MutedGroups: make([]types.GroupNotificationMute, 0)}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(BOOL_OR(show_details), FALSE)
		FROM web_push_subscription WHERE user_id = $1`, userID).Scan(&settings.Subscription, &settings.ShowDetails); err != nil {
		return types.WebPushSettings{}, err
	}
	settings.Enabled = settings.Subscription > 0
	rows, err := s.db.QueryContext(ctx, `
		SELECT groups.id, groups.group_name, preference.muted
		FROM group_notification_preference preference
		JOIN groups ON groups.id = preference.group_id
		JOIN group_member member ON member.group_id = groups.id AND member.user_id = preference.user_id
		WHERE preference.user_id = $1 AND preference.muted IS TRUE
		ORDER BY groups.group_name, groups.id`, userID)
	if err != nil {
		return types.WebPushSettings{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var mute types.GroupNotificationMute
		if err := rows.Scan(&mute.GroupID, &mute.GroupName, &mute.Muted); err != nil {
			return types.WebPushSettings{}, err
		}
		settings.MutedGroups = append(settings.MutedGroups, mute)
	}
	return settings, rows.Err()
}

func (s *Store) GetPendingDeliveries(ctx context.Context, limit int) ([]types.WebPushDelivery, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT delivery.id, subscription.id, subscription.user_id, subscription.endpoint,
			subscription.p256dh_key, subscription.auth_key, subscription.show_details,
			delivery.expense_id, delivery.recipient_user_id, delivery.group_id, delivery.actor_user_id,
			delivery.group_name, delivery.currency, delivery.amount::text, delivery.attempts, delivery.expires_at
		FROM web_push_delivery delivery
		JOIN web_push_subscription subscription
			ON subscription.id = delivery.subscription_id AND subscription.user_id = delivery.recipient_user_id
		JOIN group_member member ON member.group_id = delivery.group_id AND member.user_id = delivery.recipient_user_id
		JOIN groups ON groups.id = delivery.group_id AND groups.is_active IS TRUE
		LEFT JOIN group_notification_preference preference
			ON preference.group_id = delivery.group_id AND preference.user_id = delivery.recipient_user_id
		WHERE delivery.completed_at IS NULL
			AND delivery.available_at <= NOW()
			AND delivery.expires_at > NOW()
			AND delivery.recipient_user_id <> delivery.actor_user_id
			AND COALESCE(preference.muted, FALSE) IS FALSE
		ORDER BY delivery.available_at, delivery.id
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deliveries := make([]types.WebPushDelivery, 0, limit)
	for rows.Next() {
		var delivery types.WebPushDelivery
		if err := rows.Scan(
			&delivery.ID, &delivery.Subscription.ID, &delivery.Subscription.UserID, &delivery.Subscription.Endpoint,
			&delivery.Subscription.P256DH, &delivery.Subscription.Auth, &delivery.Subscription.ShowDetails,
			&delivery.ExpenseID, &delivery.RecipientID, &delivery.GroupID, &delivery.ActorID,
			&delivery.GroupName, &delivery.Currency, &delivery.Amount, &delivery.Attempts, &delivery.ExpiresAt,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries, rows.Err()
}

func (s *Store) CompleteDelivery(ctx context.Context, id uuid.UUID, code string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE web_push_delivery
		SET completed_at = NOW(), delivered_at = CASE WHEN $2 = 'delivered' THEN NOW() ELSE NULL END,
			failure_code = NULLIF($2, '')
		WHERE id = $1`, id, code)
	return err
}

func (s *Store) RetryDelivery(ctx context.Context, id uuid.UUID, availableAt time.Time, attempts int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE web_push_delivery
		SET attempts = $2, available_at = $3
		WHERE id = $1 AND completed_at IS NULL`, id, attempts, availableAt)
	return err
}

func (s *Store) RetireSubscription(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM web_push_subscription WHERE id = $1", id)
	return err
}

func (s *Store) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE web_push_delivery SET completed_at = NOW(), failure_code = 'expired'
		WHERE completed_at IS NULL AND expires_at <= NOW();
		DELETE FROM web_push_delivery
		WHERE completed_at < NOW() - INTERVAL '24 hours'`)
	return err
}
