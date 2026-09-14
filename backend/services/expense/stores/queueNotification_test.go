package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueueExpenseCreatedNotificationsTargetsPartialUniqueIndex(t *testing.T) {
	normalizedQuery := strings.Join(strings.Fields(queueExpenseCreatedNotificationsQuery), " ")

	require.Contains(t, normalizedQuery,
		"ON CONFLICT (expense_id, subscription_id) WHERE notification_type = 'expense_created' DO NOTHING")
}
