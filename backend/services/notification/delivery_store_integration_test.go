package notification

import (
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeliveryLeaseLifecycle(t *testing.T) {
	database := openTestDB(t)
	defer database.Close()
	ctx := t.Context()
	actor, recipient, group, expense, category := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, insertUser(ctx, database, actor))
	require.NoError(t, insertUser(ctx, database, recipient))
	require.NoError(t, insertGroupMember(ctx, database, group, actor))
	defer func() {
		_, err := database.Exec("DELETE FROM expense WHERE id = $1", expense)
		require.NoError(t, err)
		cleanupNotificationFixtures(t, database, actor, group)
		_, err = database.Exec("DELETE FROM users WHERE id = $1", recipient)
		require.NoError(t, err)
		_, err = database.Exec("DELETE FROM expense_type WHERE id = $1", category)
		require.NoError(t, err)
	}()
	_, err := database.Exec(`INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)`, uuid.New(), group, recipient)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO expense_type (id, name, category) VALUES ($1, $2, 'test')`, category, "lease-"+category.String()[:8])
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO expense (id, description, group_id, create_by_user_id, pay_by_user_id,
		exp_type_id, is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc, split_rule, occurred_on)
		VALUES ($1, 'lease test', $2, $3, $3, $4, FALSE, 1, 0, 1, 'CAD', NOW(), 'Equally', CURRENT_DATE)`, expense, group, actor, category)
	require.NoError(t, err)
	store := NewStore(database)
	endpoint := "https://fcm.googleapis.com/push/" + uuid.NewString()
	require.NoError(t, store.UpsertSubscription(ctx, recipient, types.WebPushSubscriptionInput{Endpoint: endpoint, P256DH: "test", Auth: "test"}))
	id := uuid.New()
	_, err = database.Exec(`INSERT INTO web_push_delivery (id, expense_id, subscription_id, recipient_user_id,
		group_id, actor_user_id, group_name, currency, amount, expires_at)
		SELECT $1, $2, id, $3, $4, $5, 'lease test', 'CAD', 1, NOW() + INTERVAL '1 hour'
		FROM web_push_subscription WHERE endpoint = $6`, id, expense, recipient, group, actor, endpoint)
	require.NoError(t, err)

	// A locked delivery must be skipped rather than blocking another claimer.
	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "SELECT id FROM web_push_delivery WHERE id = $1 FOR UPDATE", id)
	require.NoError(t, err)
	batch, err := store.ClaimDeliveries(ctx)
	require.NoError(t, err)
	require.Empty(t, batch.Deliveries)
	require.NoError(t, tx.Rollback())

	first, err := store.ClaimDeliveries(ctx)
	require.NoError(t, err)
	require.Len(t, first.Deliveries, 1)
	second, err := store.ClaimDeliveries(ctx)
	require.NoError(t, err)
	require.Empty(t, second.Deliveries)

	// Expired leases can be reclaimed; an old sender cannot overwrite the new result.
	_, err = database.Exec("UPDATE web_push_delivery SET claimed_until = NOW() - INTERVAL '1 second' WHERE id = $1", id)
	require.NoError(t, err)
	second, err = store.ClaimDeliveries(ctx)
	require.NoError(t, err)
	require.Len(t, second.Deliveries, 1)
	require.NotEqual(t, first.Token, second.Token)
	require.NoError(t, store.AcknowledgeDeliveries(ctx, first.Token, []DeliveryResult{{ID: id, Status: "delivered"}}))
	var completed bool
	require.NoError(t, database.QueryRow("SELECT completed_at IS NOT NULL FROM web_push_delivery WHERE id = $1", id).Scan(&completed))
	require.False(t, completed)

	result := DeliveryResult{ID: id, Status: "retry", RetryAt: time.Now().Add(time.Minute)}
	require.NoError(t, store.AcknowledgeDeliveries(ctx, second.Token, []DeliveryResult{result}))
	require.NoError(t, store.AcknowledgeDeliveries(ctx, second.Token, []DeliveryResult{result}))
	var attempts int
	require.NoError(t, database.QueryRow("SELECT attempts FROM web_push_delivery WHERE id = $1", id).Scan(&attempts))
	require.Equal(t, 1, attempts, "duplicate acknowledgements must not increment attempts")
	batch, err = store.ClaimDeliveries(ctx)
	require.NoError(t, err)
	require.Empty(t, batch.Deliveries, "retry is not due yet")
	_, err = database.Exec("UPDATE web_push_delivery SET available_at = NOW() WHERE id = $1", id)
	require.NoError(t, err)
	require.NoError(t, store.SetGroupMute(ctx, recipient, group, true))
	batch, err = store.ClaimDeliveries(ctx)
	require.NoError(t, err)
	require.Empty(t, batch.Deliveries, "mute is checked on every claim")
	require.NoError(t, store.SetGroupMute(ctx, recipient, group, false))
	batch, err = store.ClaimDeliveries(ctx)
	require.NoError(t, err)
	require.Len(t, batch.Deliveries, 1)
	require.NoError(t, store.AcknowledgeDeliveries(ctx, batch.Token, []DeliveryResult{{ID: id, Status: "retired"}}))
	var count int
	require.NoError(t, database.QueryRow("SELECT COUNT(*) FROM web_push_subscription WHERE endpoint = $1", endpoint).Scan(&count))
	require.Zero(t, count)
}
