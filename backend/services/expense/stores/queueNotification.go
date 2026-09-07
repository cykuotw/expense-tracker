package store

import "expense-tracker/backend/types"

const maxPendingWebPushDeliveries = 1000

// QueueExpenseCreatedNotifications captures currently eligible devices inside
// the expense transaction. Replayed idempotency keys never reach this method.
func (s *Store) QueueExpenseCreatedNotifications(expense types.Expense) error {
	_, err := s.db.Exec(`
		WITH capacity AS (
			SELECT GREATEST($6 - COUNT(*), 0)::bigint AS remaining
			FROM web_push_delivery
			WHERE completed_at IS NULL AND expires_at > NOW()
		), recipients AS (
			SELECT subscription.id, subscription.user_id, groups.group_name
			FROM web_push_subscription subscription
			JOIN group_member member
				ON member.user_id = subscription.user_id
			JOIN groups ON groups.id = member.group_id AND groups.is_active IS TRUE
			LEFT JOIN group_notification_preference preference
				ON preference.group_id = member.group_id
				AND preference.user_id = member.user_id
			WHERE member.group_id = $1
				AND subscription.user_id <> $2
				AND COALESCE(preference.muted, FALSE) IS FALSE
			ORDER BY subscription.id
			LIMIT (SELECT remaining FROM capacity)
		)
		INSERT INTO web_push_delivery (
			expense_id, subscription_id, recipient_user_id, group_id, actor_user_id,
			group_name, currency, amount, expires_at
		)
		SELECT $3, id, user_id, $1, $2, group_name, $4, $5, NOW() + INTERVAL '15 minutes'
		FROM recipients
		ON CONFLICT (expense_id, subscription_id) DO NOTHING`,
		expense.GroupID,
		expense.CreateByUserID,
		expense.ID,
		expense.Currency,
		expense.Total.String(),
		maxPendingWebPushDeliveries,
	)
	return err
}
