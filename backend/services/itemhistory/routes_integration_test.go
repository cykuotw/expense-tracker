package itemhistory

import (
	"encoding/json"
	"expense-tracker/backend/internal/ocrtest"
	"expense-tracker/backend/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestHistoryAuthorizationPagingCurrenciesAndLegacyDates(t *testing.T) {
	f := ocrtest.New(t)
	s := Store{f.DB}
	for range 24 {
		f.InsertExpense(t, uuid.New(), "CAD", "Milk 2%")
	}
	f.InsertExpense(t, uuid.New(), "USD", "Milk imported")
	_, err := f.DB.Exec(`UPDATE expense SET occurred_on=NULL WHERE id=$1`, f.Expense)
	require.NoError(t, err)
	seen := map[uuid.UUID]bool{}
	cursor := ""
	currencies := map[string]bool{}
	for range 3 {
		page, err := s.List(t.Context(), f.Group, f.User, "milk", cursor)
		require.NoError(t, err)
		require.LessOrEqual(t, len(page.Items), PageSize)
		for _, row := range page.Items {
			require.False(t, seen[row.ItemID])
			seen[row.ItemID] = true
			currencies[row.Currency] = true
			require.Equal(t, "2022-03-18", row.Date)
			require.Nil(t, row.Quantity)
			require.Nil(t, row.UnitPrice)
			require.Equal(t, "5.390", row.LineTotal)
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	require.Len(t, seen, 26)
	require.True(t, currencies["CAD"] && currencies["USD"])
	_, err = s.List(t.Context(), f.Group, f.Other, "milk", "")
	require.ErrorIs(t, err, types.ErrGroupNotExist)
	page, err := s.List(t.Context(), f.Group, f.User, "milk", "")
	require.NoError(t, err)
	_, err = s.List(t.Context(), f.Group, f.User, "other", page.NextCursor)
	require.ErrorIs(t, err, ErrCursor)
	_, err = s.List(t.Context(), f.Group, f.User, strings.Repeat("乳", 80), "")
	require.NoError(t, err)
	_, err = s.List(t.Context(), f.Group, f.User, strings.Repeat("x", 201), "")
	require.ErrorIs(t, err, ErrQuery)
	_, err = f.DB.Exec(`UPDATE expense SET is_deleted=TRUE WHERE group_id=$1`, f.Group)
	require.NoError(t, err)
	page, err = s.List(t.Context(), f.Group, f.User, "milk", "")
	require.NoError(t, err)
	require.Empty(t, page.Items)
}

func TestHistoryLegacyProjectionAndMeasuredPlan(t *testing.T) {
	f := ocrtest.New(t)
	s := Store{f.DB}
	_, err := f.DB.Exec(`UPDATE item SET description=NULL,amount=2,unit_price=2.695,line_total=NULL WHERE expense_id=$1`, f.Expense)
	require.NoError(t, err)
	page, err := s.List(t.Context(), f.Group, f.User, "milk", "")
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "5.390", page.Items[0].LineTotal)
	require.NotNil(t, page.Items[0].Quantity)
	require.True(t, decimal.RequireFromString(*page.Items[0].Quantity).Equal(decimal.NewFromInt(2)))
	require.NotNil(t, page.Items[0].UnitPrice)
	// A thousand persisted purchases exercise the actual search plan using the
	// existing expense/group and item/expense indexes, not a synthetic query.
	_, err = f.DB.Exec(`INSERT INTO expense(id,description,group_id,create_by_user_id,pay_by_user_id,exp_type_id,is_settled,sub_total,tax_fee_tip,total,currency,create_time_utc,expense_time_utc,allocation_mode,is_deleted,occurred_on,provider_name) SELECT gen_random_uuid(),'History plan',$1,$2,$2,$3,FALSE,5.39,0,5.39,'CAD',NOW(),NOW(),'equal',FALSE,'2022-03-18','Shop' FROM generate_series(1,1000)`, f.Group, f.User, f.Category)
	require.NoError(t, err)
	_, err = f.DB.Exec(`INSERT INTO item(id,expense_id,name,amount,description,line_total,position,unit_price) SELECT gen_random_uuid(),id,'Milk',1,'Milk',5.39,0,5.39 FROM expense WHERE group_id=$1 AND id<>$2`, f.Group, f.Expense)
	require.NoError(t, err)
	var plan json.RawMessage
	maxID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	err = f.DB.QueryRowContext(t.Context(), `EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) `+historyQuery, f.Group, "milk", f.User, "9999-12-31", maxID, maxID).Scan(&plan)
	require.NoError(t, err)
	var parsed []struct {
		Plan struct {
			ActualRows int `json:"Actual Rows"`
		} `json:"Plan"`
		ExecutionTime float64 `json:"Execution Time"`
	}
	require.NoError(t, json.Unmarshal(plan, &parsed))
	require.Len(t, parsed, 1)
	require.Equal(t, 21, parsed[0].Plan.ActualRows)
	t.Logf("item history EXPLAIN ANALYZE: 1001 purchases, root rows=%d, execution_ms=%.3f", parsed[0].Plan.ActualRows, parsed[0].ExecutionTime)
	page, err = s.List(t.Context(), f.Group, f.User, "milk", "")
	require.NoError(t, err)
	require.Len(t, page.Items, 20)
}
