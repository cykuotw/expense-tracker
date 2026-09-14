package monthlyreview

import (
	"database/sql"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/types"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestConcurrentPublicationDoesNotDuplicateMarkerOrDeliveries(t *testing.T) {
	database := openMonthlyReviewTestDB(t)
	fixture := createMonthlyReviewFixture(t, database)
	now := time.Date(1900, time.September, 13, 10, 0, 0, 0, time.UTC)

	errors := make(chan error, 2)
	var publishers sync.WaitGroup
	for range 2 {
		publishers.Go(func() {
			_, err := NewStore(database).PublishEligible(t.Context(), now, 25)
			errors <- err
		})
	}
	publishers.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	var publications, deliveries int
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM monthly_review_publication
		WHERE group_id = $1 AND review_month = '1900-08-01'`, fixture.homeGroup).Scan(&publications))
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM web_push_delivery
		WHERE group_id = $1 AND review_month = '1900-08-01'
		AND notification_type = 'monthly_review_available'`, fixture.homeGroup).Scan(&deliveries))
	require.Equal(t, 1, publications)
	require.Equal(t, 2, deliveries)
}

func TestTrendReturnsTwelvePublishedMonthsAndLatestReport(t *testing.T) {
	database := openMonthlyReviewTestDB(t)
	fixture := createMonthlyReviewFixture(t, database)
	store := NewStore(database)
	now := time.Date(2027, time.September, 13, 10, 0, 0, 0, time.UTC)
	august := time.Date(2027, time.August, 1, 0, 0, 0, 0, time.UTC)
	_, err := database.Exec("UPDATE expense SET occurred_on = '2027-08-20' WHERE id = $1", fixture.expense)
	require.NoError(t, err)

	_, err = database.Exec(`INSERT INTO monthly_review_publication (group_id, review_month)
		VALUES ($1, $2)`, fixture.homeGroup, august)
	require.NoError(t, err)

	trend, err := store.GetTrend(t.Context(), fixture.homeGroup, fixture.alice, august, now)
	require.NoError(t, err)
	require.Equal(t, "2026-09", trend.StartMonth)
	require.Equal(t, "2027-08", trend.EndMonth)
	require.Equal(t, "2027-08", trend.LatestReportMonth)
	require.Len(t, trend.Currencies, 1)
	require.Equal(t, "CAD", trend.Currencies[0].Currency)
	require.Len(t, trend.Currencies[0].Months, 12)
	require.Equal(t, TrendMonth{Month: "2026-09", Total: "0"}, trend.Currencies[0].Months[0])
	require.Equal(t, TrendMonth{Month: "2027-08", Total: "30", ExpenseCount: 1}, trend.Currencies[0].Months[11])

	trend, err = store.GetTrend(t.Context(), fixture.homeGroup, fixture.alice, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), now)
	require.NoError(t, err)
	require.Equal(t, "2026-01", trend.StartMonth)
	require.Equal(t, "2026-02", trend.EndMonth)

	_, err = store.GetTrend(t.Context(), fixture.homeGroup, uuid.New(), august, now)
	require.ErrorIs(t, err, types.ErrGroupNotExist)
}

func TestPublicationAndLiveReviewLifecycle(t *testing.T) {
	database := openMonthlyReviewTestDB(t)
	fixture := createMonthlyReviewFixture(t, database)
	store := NewStore(database)
	now := time.Date(1900, time.September, 13, 10, 0, 0, 0, time.UTC)
	august := time.Date(1900, time.August, 1, 0, 0, 0, 0, time.UTC)

	review, err := store.Get(t.Context(), fixture.homeGroup, fixture.alice, august, now)
	require.NoError(t, err)
	require.Equal(t, StateUnpublished, review.State)

	result, err := store.PublishEligible(t.Context(), now, 25)
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.Published, 1)
	require.GreaterOrEqual(t, result.Deliveries, 2)

	result, err = store.PublishEligible(t.Context(), now, 25)
	require.NoError(t, err)
	require.Zero(t, result.Published)
	require.Zero(t, result.Deliveries)

	var publicationCount, deliveryCount int
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM monthly_review_publication
		WHERE group_id = $1 AND review_month = $2`, fixture.homeGroup, august).Scan(&publicationCount))
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM web_push_delivery
		WHERE group_id = $1 AND review_month = $2 AND notification_type = 'monthly_review_available'`, fixture.homeGroup, august).Scan(&deliveryCount))
	require.Equal(t, 1, publicationCount)
	require.Equal(t, 2, deliveryCount)
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM monthly_review_publication
		WHERE group_id = $1`, fixture.tripGroup).Scan(&publicationCount))
	require.Zero(t, publicationCount, "non-Home groups must never publish reviews")

	review, err = store.Get(t.Context(), fixture.homeGroup, fixture.alice, august, now)
	require.NoError(t, err)
	require.Equal(t, StatePublished, review.State)
	require.Len(t, review.Currencies, 1)
	require.Equal(t, "CAD", review.Currencies[0].Currency)
	require.Equal(t, "30", review.Currencies[0].Total)
	require.Equal(t, 1, review.Currencies[0].ExpenseCount)
	require.Len(t, review.Currencies[0].MemberNetTotals, 2)
	expensePage, err := store.ListExpenses(t.Context(), fixture.homeGroup, fixture.alice, august, "CAD", "")
	require.NoError(t, err)
	require.Len(t, expensePage.Expenses, 1)
	require.Empty(t, expensePage.NextCursor)

	_, err = store.Get(t.Context(), fixture.homeGroup, uuid.New(), august, now)
	require.ErrorIs(t, err, types.ErrGroupNotExist)
	current, err := store.Get(t.Context(), fixture.homeGroup, fixture.alice, firstOfMonth(now), now)
	require.NoError(t, err)
	require.Equal(t, StateNotClosed, current.State)

	_, err = database.Exec("UPDATE expense SET total = 45, is_settled = TRUE WHERE id = $1", fixture.expense)
	require.NoError(t, err)
	_, err = database.Exec("UPDATE ledger SET share = 22.5 WHERE expense_id = $1", fixture.expense)
	require.NoError(t, err)
	review, err = store.Get(t.Context(), fixture.homeGroup, fixture.alice, august, now)
	require.NoError(t, err)
	require.Equal(t, "45", review.Currencies[0].Total)
	expensePage, err = store.ListExpenses(t.Context(), fixture.homeGroup, fixture.alice, august, "CAD", "")
	require.NoError(t, err)
	require.True(t, expensePage.Expenses[0].Settled)

	_, err = database.Exec("UPDATE expense SET occurred_on = '1900-07-20' WHERE id = $1", fixture.expense)
	require.NoError(t, err)
	review, err = store.Get(t.Context(), fixture.homeGroup, fixture.alice, august, now)
	require.NoError(t, err)
	require.Equal(t, StateEmpty, review.State)

	result, err = store.PublishEligible(t.Context(), now, 25)
	require.NoError(t, err)
	require.Equal(t, 1, result.Published)
	require.Equal(t, 2, result.Deliveries)
	july := time.Date(1900, time.July, 1, 0, 0, 0, 0, time.UTC)
	review, err = store.Get(t.Context(), fixture.homeGroup, fixture.alice, july, now)
	require.NoError(t, err)
	require.Equal(t, StatePublished, review.State)

	_, err = database.Exec("UPDATE expense SET is_deleted = TRUE WHERE id = $1", fixture.expense)
	require.NoError(t, err)
	review, err = store.Get(t.Context(), fixture.homeGroup, fixture.alice, july, now)
	require.NoError(t, err)
	require.Equal(t, StateEmpty, review.State)
	_, err = database.Exec("UPDATE expense SET is_deleted = FALSE WHERE id = $1", fixture.expense)
	require.NoError(t, err)
	review, err = store.Get(t.Context(), fixture.homeGroup, fixture.alice, july, now)
	require.NoError(t, err)
	require.Equal(t, StatePublished, review.State)
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM web_push_delivery
		WHERE group_id = $1 AND review_month = $2 AND notification_type = 'monthly_review_available'`, fixture.homeGroup, july).Scan(&deliveryCount))
	require.Equal(t, 2, deliveryCount, "live changes must not enqueue another availability event")
}

func TestMonthlyReviewExpensesUseBoundedCursorPagination(t *testing.T) {
	database := openMonthlyReviewTestDB(t)
	fixture := createMonthlyReviewFixture(t, database)
	store := NewStore(database)
	now := time.Date(1900, time.September, 13, 10, 0, 0, 0, time.UTC)
	august := time.Date(1900, time.August, 1, 0, 0, 0, 0, time.UTC)

	for index := range 21 {
		_, err := database.Exec(`INSERT INTO expense (
			id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id,
			is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc,
			expense_time_utc, allocation_mode, is_deleted, occurred_on
		) VALUES ($1, $2, $3, $4, $4, $5, FALSE, 1, 0, 1, 'CAD', NOW(), $6, 'equal', FALSE, $7)`,
			uuid.New(), fmt.Sprintf("Paginated expense %02d", index), fixture.homeGroup,
			fixture.alice, fixture.category, now.Add(-time.Duration(index)*time.Minute),
			fmt.Sprintf("1900-08-%02d", index%20+1))
		require.NoError(t, err)
	}
	_, err := store.PublishEligible(t.Context(), now, 25)
	require.NoError(t, err)

	firstPage, err := store.ListExpenses(t.Context(), fixture.homeGroup, fixture.alice, august, "CAD", "")
	require.NoError(t, err)
	require.Len(t, firstPage.Expenses, monthlyReviewExpensePageSize)
	require.NotEmpty(t, firstPage.NextCursor)

	secondPage, err := store.ListExpenses(t.Context(), fixture.homeGroup, fixture.alice, august, "CAD", firstPage.NextCursor)
	require.NoError(t, err)
	require.Len(t, secondPage.Expenses, 2)
	require.Empty(t, secondPage.NextCursor)
	firstIDs := make(map[string]struct{}, len(firstPage.Expenses))
	for _, expense := range firstPage.Expenses {
		firstIDs[expense.ID] = struct{}{}
	}
	for _, expense := range secondPage.Expenses {
		_, duplicate := firstIDs[expense.ID]
		require.False(t, duplicate)
	}

	_, err = store.ListExpenses(t.Context(), fixture.homeGroup, fixture.alice, august, "CAD", "not-a-cursor")
	require.ErrorIs(t, err, ErrInvalidExpenseCursor)
	_, err = store.ListExpenses(t.Context(), fixture.homeGroup, uuid.New(), august, "CAD", "")
	require.ErrorIs(t, err, types.ErrGroupNotExist)
}

type monthlyReviewFixture struct {
	alice, bob, homeGroup, tripGroup, expense, tripExpense, category uuid.UUID
}

func openMonthlyReviewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	require.NoError(t, err)
	require.NoError(t, database.Ping())
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func createMonthlyReviewFixture(t *testing.T, database *sql.DB) monthlyReviewFixture {
	t.Helper()
	fixture := monthlyReviewFixture{alice: uuid.New(), bob: uuid.New(), homeGroup: uuid.New(), tripGroup: uuid.New(), expense: uuid.New(), tripExpense: uuid.New(), category: uuid.New()}
	for _, userID := range []uuid.UUID{fixture.alice, fixture.bob} {
		_, err := database.Exec(`INSERT INTO users (id, username, firstname, lastname, email, password_hash, create_time_utc, is_active, has_local_password, role)
			VALUES ($1, $2, 'Review', 'Test', $3, 'unused', NOW(), TRUE, TRUE, 'user')`,
			userID, "review-"+userID.String()[:8], userID.String()+"@example.test")
		require.NoError(t, err)
	}
	for _, group := range []struct {
		id, creator uuid.UUID
		groupType   string
	}{
		{fixture.homeGroup, fixture.alice, "home"},
		{fixture.tripGroup, fixture.alice, "trip"},
	} {
		_, err := database.Exec(`INSERT INTO groups (id, group_name, description, create_time_utc, is_active, create_by_user_id, currency, group_type)
			VALUES ($1, $2, '', NOW(), TRUE, $3, 'CAD', $4)`, group.id, "review-"+group.id.String()[:8], group.creator, group.groupType)
		require.NoError(t, err)
		_, err = database.Exec("INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)", uuid.New(), group.id, fixture.alice)
		require.NoError(t, err)
	}
	_, err := database.Exec("INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)", uuid.New(), fixture.homeGroup, fixture.bob)
	require.NoError(t, err)
	_, err = database.Exec("INSERT INTO expense_type (id, name, category) VALUES ($1, 'Groceries', 'Food and Drink')", fixture.category)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO expense (
		id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id,
		is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc,
		expense_time_utc, allocation_mode, is_deleted, occurred_on
	) VALUES ($1, 'Trip expense', $2, $3, $3, $4, FALSE, 10, 0, 10, 'CAD', NOW(), NOW(), 'equal', FALSE, '1900-08-21')`,
		fixture.tripExpense, fixture.tripGroup, fixture.alice, fixture.category)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO expense (
		id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id,
		is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc,
		expense_time_utc, allocation_mode, is_deleted, occurred_on
	) VALUES ($1, 'August groceries', $2, $3, $3, $4, FALSE, 30, 0, 30, 'CAD', NOW(), NOW(), 'equal', FALSE, '1900-08-20')`,
		fixture.expense, fixture.homeGroup, fixture.alice, fixture.category)
	require.NoError(t, err)
	_, err = database.Exec("INSERT INTO item (id, expense_id, name, amount, unit, unit_price) VALUES ($1, $2, 'Milk', 1, 'each', 5)", uuid.New(), fixture.expense)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO ledger (id, expense_id, lender_user_id, borrower_user_id, share)
		VALUES ($1, $2, $3, $4, 15)`, uuid.New(), fixture.expense, fixture.alice, fixture.bob)
	require.NoError(t, err)

	for _, userID := range []uuid.UUID{fixture.alice, fixture.bob} {
		_, err = database.Exec(`INSERT INTO web_push_subscription (id, user_id, endpoint, p256dh_key, auth_key)
			VALUES ($1, $2, $3, 'test', 'test')`, uuid.New(), userID, "https://fcm.googleapis.com/push/"+uuid.NewString())
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		_, err := database.Exec("DELETE FROM expense WHERE group_id = ANY($1)", []uuid.UUID{fixture.homeGroup, fixture.tripGroup})
		require.NoError(t, err)
		_, err = database.Exec("DELETE FROM groups WHERE id = ANY($1)", []uuid.UUID{fixture.homeGroup, fixture.tripGroup})
		require.NoError(t, err)
		_, err = database.Exec("DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{fixture.alice, fixture.bob})
		require.NoError(t, err)
		_, err = database.Exec("DELETE FROM expense_type WHERE id = $1", fixture.category)
		require.NoError(t, err)
	})
	return fixture
}
