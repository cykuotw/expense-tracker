package notification

import (
	"context"
	"database/sql"
	"errors"
	"expense-tracker/backend/types"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// These contracts are only exposed through IAM-authorized Lambda invocation.
type DeliveryRequest struct {
	Action  string           `json:"action"`
	Token   uuid.UUID        `json:"token"`
	Results []DeliveryResult `json:"results,omitempty"`
}

type DeliveryBatch struct {
	Token      uuid.UUID               `json:"token"`
	Deliveries []types.WebPushDelivery `json:"deliveries"`
}

type DeliveryResult struct {
	ID      uuid.UUID `json:"id"`
	Status  string    `json:"status"`
	RetryAt time.Time `json:"retryAt,omitzero"`
}

func (s *Store) HandleDeliveryRequest(ctx context.Context, request DeliveryRequest) (DeliveryBatch, error) {
	switch request.Action {
	case "claim":
		return s.ClaimDeliveries(ctx)
	case "ack":
		return DeliveryBatch{}, s.AcknowledgeDeliveries(ctx, request.Token, request.Results)
	default:
		return DeliveryBatch{}, errors.New("unsupported delivery operation")
	}
}

func (s *Store) ClaimDeliveries(ctx context.Context) (DeliveryBatch, error) {
	if err := s.Cleanup(ctx); err != nil {
		return DeliveryBatch{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return DeliveryBatch{}, err
	}
	defer tx.Rollback()
	deliveries, err := pendingDeliveries(ctx, tx, senderBatchSize)
	if err != nil {
		return DeliveryBatch{}, err
	}
	batch := DeliveryBatch{Token: uuid.New(), Deliveries: deliveries}
	for _, delivery := range deliveries {
		if _, err := tx.ExecContext(ctx, `UPDATE web_push_delivery
			SET claim_token = $2, claimed_until = NOW() + INTERVAL '2 minutes'
			WHERE id = $1`, delivery.ID, batch.Token); err != nil {
			return DeliveryBatch{}, err
		}
	}
	return batch, tx.Commit()
}

func (s *Store) AcknowledgeDeliveries(ctx context.Context, token uuid.UUID, results []DeliveryResult) error {
	if token == uuid.Nil || len(results) > senderBatchSize {
		return errors.New("invalid delivery acknowledgement")
	}
	for _, result := range results {
		switch result.Status {
		case "delivered", "provider_rejected", "failed", "retired":
		case "retry":
			if result.RetryAt.IsZero() {
				return errors.New("missing retry time")
			}
		default:
			return errors.New("invalid delivery result")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, result := range results {
		var subscriptionID uuid.UUID
		var attempts int
		var expiresAt time.Time
		err := tx.QueryRowContext(ctx, `SELECT subscription_id, attempts, expires_at
			FROM web_push_delivery WHERE id = $1 AND claim_token = $2
			AND claimed_until > NOW() AND completed_at IS NULL FOR UPDATE`, result.ID, token).
			Scan(&subscriptionID, &attempts, &expiresAt)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		} // Duplicate or stale acknowledgement.
		if err != nil {
			return err
		}
		if result.Status == "retired" {
			_, err = tx.ExecContext(ctx, "DELETE FROM web_push_subscription WHERE id = $1", subscriptionID)
		} else if result.Status == "retry" && attempts+1 < maxAttempts && result.RetryAt.Before(expiresAt) {
			_, err = tx.ExecContext(ctx, `UPDATE web_push_delivery
				SET attempts = attempts + 1, available_at = GREATEST($2, NOW() + INTERVAL '1 second'),
				claim_token = NULL, claimed_until = NULL WHERE id = $1`, result.ID, result.RetryAt)
		} else {
			status := result.Status
			if status == "retry" {
				status = "failed"
			}
			_, err = tx.ExecContext(ctx, `UPDATE web_push_delivery SET completed_at = NOW(),
				delivered_at = CASE WHEN $2 = 'delivered' THEN NOW() ELSE NULL END,
				failure_code = $2, claim_token = NULL, claimed_until = NULL WHERE id = $1`, result.ID, status)
		}
		if err != nil {
			return fmt.Errorf("acknowledge delivery: %w", err)
		}
	}
	return tx.Commit()
}

type deliveryQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func pendingDeliveries(ctx context.Context, db deliveryQueryer, limit int) ([]types.WebPushDelivery, error) {
	query := `
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
			AND (delivery.claimed_until IS NULL OR delivery.claimed_until <= NOW())
			AND delivery.recipient_user_id <> delivery.actor_user_id
			AND COALESCE(preference.muted, FALSE) IS FALSE
		ORDER BY delivery.available_at, delivery.id
		LIMIT $1`
	query += " FOR UPDATE OF delivery SKIP LOCKED"
	rows, err := db.QueryContext(ctx, query, limit)
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

func (s *Store) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE web_push_delivery SET completed_at = NOW(), failure_code = 'expired'
		WHERE completed_at IS NULL AND expires_at <= NOW();
		DELETE FROM web_push_delivery
		WHERE completed_at < NOW() - INTERVAL '7 days'`)
	return err
}
