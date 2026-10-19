package monthlyreview

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCategoryBreakdownIntegrationTotalsAndDrilldown(t *testing.T) {
	database := openMonthlyReviewTestDB(t)
	fixture := createMonthlyReviewFixture(t, database)
	homeType, otherFoodType := uuid.New(), uuid.New()
	_, err := database.Exec(`INSERT INTO expense_type(id, name, category) VALUES ($1, 'Household Supplies', 'Home'), ($2, 'Other food', 'Food and Drink')`, homeType, otherFoodType)
	require.NoError(t, err)
	defer func() {
		_, err := database.Exec(`DELETE FROM expense_category_breakdown WHERE expense_id IN (SELECT id FROM expense WHERE group_id = $1)`, fixture.homeGroup)
		require.NoError(t, err)
		_, err = database.Exec(`DELETE FROM expense_type WHERE id IN ($1, $2)`, homeType, otherFoodType)
		require.NoError(t, err)
	}()
	_, err = database.Exec(`UPDATE expense SET total = 300, occurred_on = '1900-08-20' WHERE id = $1`, fixture.expense)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO expense_category_breakdown (expense_id, position, expense_type_id, amount) VALUES ($1, 0, $2, 150), ($1, 1, $3, 50), ($1, 2, $4, 100)`, fixture.expense, fixture.category, otherFoodType, homeType)
	require.NoError(t, err)
	month := time.Date(1900, time.August, 1, 0, 0, 0, 0, time.UTC)
	_, err = database.Exec(`INSERT INTO monthly_review_publication (group_id, review_month) VALUES ($1, $2)`, fixture.homeGroup, month)
	require.NoError(t, err)
	store := NewStore(database)
	summaries, err := store.currencySummaries(t.Context(), fixture.homeGroup, month)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, "300", summaries[0].Total)
	require.Equal(t, 1, summaries[0].ExpenseCount)
	require.Equal(t, []NamedAmount{{Name: "Food and Drink", Amount: "200"}, {Name: "Home", Amount: "100"}}, summaries[0].Categories)
	require.Len(t, summaries[0].Payers, 1)
	require.Equal(t, "300", summaries[0].Payers[0].Amount)
	for _, category := range []struct{ name, amount string }{{"Food and Drink", "200"}, {"Home", "100"}} {
		page, err := store.ListExpenses(t.Context(), fixture.homeGroup, fixture.alice, month, "CAD", "", ExpenseListOptions{Category: category.name, Sort: "amount_desc"})
		require.NoError(t, err)
		require.Len(t, page.Expenses, 1)
		require.Equal(t, "300.000", page.Expenses[0].Total)
		require.Equal(t, category.amount, page.Expenses[0].CategoryAmount)
		require.Equal(t, 3, page.Expenses[0].CategoryCount)
	}

	// Category amount order intentionally opposes the parent-total order.
	for index := range monthlyReviewExpensePageSize + 2 {
		id := uuid.New()
		_, err := database.Exec(`INSERT INTO expense (id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id, is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc, expense_time_utc, allocation_mode, occurred_on)
            SELECT $1, 'Mixed purchase', group_id, create_by_user_id, pay_by_user_id, exp_type_id, FALSE, 0, 0, $2, currency, NOW(), NOW(), allocation_mode, occurred_on FROM expense WHERE id = $3`, id, 1000+index, fixture.expense)
		require.NoError(t, err)
		homeAmount := monthlyReviewExpensePageSize + 2 - index
		_, err = database.Exec(`INSERT INTO expense_category_breakdown (expense_id, position, expense_type_id, amount) VALUES ($1, 0, $2, $3), ($1, 1, $4, $5)`, id, fixture.category, 1000+index-homeAmount, homeType, homeAmount)
		require.NoError(t, err)
	}
	seen := make(map[string]bool)
	previous := decimal.Zero
	cursor := ""
	for {
		page, err := store.ListExpenses(t.Context(), fixture.homeGroup, fixture.alice, month, "CAD", cursor, ExpenseListOptions{Category: "Home", Sort: "amount_asc"})
		require.NoError(t, err)
		for _, row := range page.Expenses {
			require.False(t, seen[row.ID])
			seen[row.ID] = true
			amount := decimal.RequireFromString(row.CategoryAmount)
			require.True(t, amount.GreaterThanOrEqual(previous))
			previous = amount
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	require.Len(t, seen, monthlyReviewExpensePageSize+3)

	_, err = store.ListExpenses(t.Context(), fixture.homeGroup, uuid.New(), month, "CAD", "", ExpenseListOptions{Category: "Home"})
	require.Error(t, err)
	_, err = database.Exec(`UPDATE expense SET is_deleted = TRUE WHERE group_id = $1`, fixture.homeGroup)
	require.NoError(t, err)
	summaries, err = store.currencySummaries(t.Context(), fixture.homeGroup, month)
	require.NoError(t, err)
	require.Empty(t, summaries)
}
