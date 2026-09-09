package expense

import (
	"encoding/json"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formOptionsGroupStore(userID, groupID uuid.UUID) *mockGroupStore {
	store := groupStoreMock()
	store.ListCurrenciesFn = func() ([]types.Currency, error) {
		return []types.Currency{{Code: "CAD", DisplayName: "Canadian Dollar", AmountDigits: 2}}, nil
	}
	store.GetGroupListByUserFn = func(string) ([]types.GetGroupListResponse, error) {
		return []types.GetGroupListResponse{{ID: groupID.String(), GroupName: "Trip", Currency: "CAD"}}, nil
	}
	store.GetGroupByIDAndUserFn = func(gotGroupID, gotUserID string) (*types.Group, error) {
		if gotGroupID != groupID.String() || gotUserID != userID.String() {
			return nil, types.ErrUserNotPermitted
		}
		return &types.Group{ID: groupID, GroupName: "Trip", Currency: "CAD"}, nil
	}
	store.GetGroupMemberByGroupIDFn = func(string) ([]*types.User, error) {
		return []*types.User{{ID: uuid.New(), Username: "Other"}, {ID: userID, Username: "Current"}}, nil
	}
	store.CanEditGroupCurrencyFn = func(string, string) (bool, error) {
		return false, nil
	}
	return store
}

func formOptionsExpenseStore(expenseTypeID uuid.UUID) *mockExpenseStore {
	store := expenseStoreMock()
	store.GetExpenseTypeFn = func() ([]*types.ExpenseType, error) {
		return []*types.ExpenseType{{ID: expenseTypeID, Name: "General", Category: "Other"}}, nil
	}
	store.GetItemsByExpenseIDFn = func(string) ([]*types.Item, error) { return []*types.Item{}, nil }
	store.GetLedgersByExpenseIDFn = func(string) ([]*types.Ledger, error) { return []*types.Ledger{}, nil }
	return store
}

func TestGetCreateExpenseOptionsReturnsOnePageReadModel(t *testing.T) {
	userID := uuid.New()
	groupID := uuid.New()
	expenseTypeID := uuid.New()
	handler := NewHandler(
		formOptionsExpenseStore(expenseTypeID),
		userStoreMock(),
		formOptionsGroupStore(userID, groupID),
		expenseControllerMock(),
	)
	context, recorder := newReadContext(nil)
	context.Request = httptest.NewRequest(http.MethodGet, "/expense_create_options?groupId="+groupID.String(), nil)
	context.Set("userID", userID.String())

	handler.handleGetCreateExpenseOptions(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response types.CreateExpenseOptionsResponse
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	require.Len(t, response.Groups, 1)
	require.Len(t, response.ExpenseTypes, 1)
	require.Len(t, response.Currencies, 1)
	assert.Equal(t, "CAD", response.Currencies[0].Code)
	require.NotNil(t, response.Group)
	assert.Equal(t, "CAD", response.Group.Currency)
	require.Len(t, response.Group.Members, 2)
	assert.Equal(t, userID.String(), response.Group.Members[1].UserID)
}

func TestGetEditExpenseOptionsReturnsOnePageReadModel(t *testing.T) {
	userID := uuid.New()
	groupID := uuid.New()
	expenseID := uuid.New()
	expenseTypeID := uuid.New()
	store := formOptionsExpenseStore(expenseTypeID)
	userStore := userStoreMock()
	userStore.GetUserByIDFn = func(string) (*types.User, error) {
		return &types.User{ID: userID, Username: "Current"}, nil
	}
	handler := NewHandler(
		store,
		userStore,
		formOptionsGroupStore(userID, groupID),
		expenseControllerMock(),
	)
	context, recorder := newReadContext(gin.Params{{Key: "expenseId", Value: expenseID.String()}})
	context.Request = httptest.NewRequest(http.MethodGet, "/expense/"+expenseID.String()+"/edit-options", nil)
	context.Set("userID", userID.String())
	context.Set("expense", &types.Expense{
		ID:             expenseID,
		GroupID:        groupID,
		CreateByUserID: userID,
		ExpenseTypeID:  expenseTypeID,
		Currency:       "CAD",
		OccurredOn:     "2026-09-08",
	})

	handler.handleGetEditExpenseOptions(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response types.EditExpenseOptionsResponse
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	assert.Equal(t, expenseID, response.Expense.ID)
	assert.Equal(t, groupID.String(), response.Expense.GroupId)
	require.Len(t, response.Groups, 1)
	require.Len(t, response.ExpenseTypes, 1)
	require.Len(t, response.Currencies, 1)
	assert.Equal(t, "CAD", response.Currencies[0].Code)
	require.Len(t, response.Group.Members, 2)
}
