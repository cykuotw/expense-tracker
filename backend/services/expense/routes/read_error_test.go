package expense

import (
	"encoding/json"
	"errors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetExpenseDetailStopsAfterRequiredReadFailure(t *testing.T) {
	testCases := []struct {
		name           string
		failAt         string
		expectedStages []string
	}{
		{name: "items", failAt: "items", expectedStages: []string{"creator", "items"}},
		{name: "ledgers", failAt: "ledgers", expectedStages: []string{"creator", "items", "ledgers"}},
		{name: "lender username", failAt: "lender", expectedStages: []string{"creator", "items", "ledgers", "lender"}},
		{name: "borrower username", failAt: "borrower", expectedStages: []string{"creator", "items", "ledgers", "lender", "borrower"}},
		{name: "expense type", failAt: "expense type", expectedStages: []string{"creator", "items", "ledgers", "lender", "borrower", "expense type"}},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			wantErr := errors.New("required read failed")
			stages := []string{}
			creatorID := uuid.New()
			lenderID := uuid.New()
			borrowerID := uuid.New()
			expenseTypeID := uuid.New()

			store := expenseStoreMock()
			store.GetItemsByExpenseIDFn = func(string) ([]*types.Item, error) {
				stages = append(stages, "items")
				if test.failAt == "items" {
					return nil, wantErr
				}
				return []*types.Item{{ID: uuid.New()}}, nil
			}
			store.GetLedgersByExpenseIDFn = func(string) ([]*types.Ledger, error) {
				stages = append(stages, "ledgers")
				if test.failAt == "ledgers" {
					return nil, wantErr
				}
				return []*types.Ledger{{
					ID:             uuid.New(),
					LenderUserID:   lenderID,
					BorrowerUesrID: borrowerID,
				}}, nil
			}
			store.GetExpenseTypeFn = func() ([]*types.ExpenseType, error) {
				stages = append(stages, "expense type")
				if test.failAt == "expense type" {
					return nil, wantErr
				}
				return []*types.ExpenseType{{ID: expenseTypeID, Name: "Dining", Category: "Food and Drink"}}, nil
			}

			userStore := userStoreMock()
			userStore.GetUserByIDFn = func(string) (*types.User, error) {
				stages = append(stages, "creator")
				return &types.User{ID: creatorID, Username: "creator"}, nil
			}
			userStore.GetUsernameByIDFn = func(id string) (string, error) {
				stage := "borrower"
				if id == lenderID.String() {
					stage = "lender"
				}
				stages = append(stages, stage)
				if test.failAt == stage {
					return "", wantErr
				}
				return stage, nil
			}

			handler := NewHandler(store, userStore, groupStoreMock(), expenseControllerMock())
			context, recorder := newReadContext(gin.Params{{Key: "expenseId", Value: uuid.NewString()}})
			context.Set("userID", uuid.NewString())
			context.Set("expense", &types.Expense{
				CreateByUserID: creatorID,
				ExpenseTypeID:  expenseTypeID,
			})

			handler.handleGetExpenseDetail(context)

			requireInternalError(t, recorder)
			require.Equal(t, test.expectedStages, stages)
		})
	}
}

func TestGetExpenseListStopsAfterCurrencyFailure(t *testing.T) {
	wantErr := errors.New("currency read failed")
	stages := []string{}
	store := expenseStoreMock()
	store.GetExpenseListFn = func(string, int64, types.ExpenseListOrder, types.ExpenseListStatus) (*types.ExpenseListPage, error) {
		stages = append(stages, "expenses")
		return &types.ExpenseListPage{Expenses: []*types.Expense{{ID: uuid.New()}}}, nil
	}
	store.GetLedgersByExpenseIDFn = func(string) ([]*types.Ledger, error) {
		stages = append(stages, "ledgers")
		return nil, nil
	}
	groupStore := groupStoreMock()
	groupStore.GetGroupCurrencyFn = func(string) (string, error) {
		stages = append(stages, "currency")
		return "", wantErr
	}
	handler := NewHandler(store, userStoreMock(), groupStore, expenseControllerMock())
	context, recorder := newReadContext(gin.Params{{Key: "groupId", Value: uuid.NewString()}})

	handler.handleGetExpenseList(context)

	requireInternalError(t, recorder)
	require.Equal(t, []string{"expenses", "currency"}, stages)
}

func TestGetUnsettledBalanceStopsAfterBalanceFailure(t *testing.T) {
	wantErr := errors.New("balance read failed")
	stages := []string{}
	store := expenseStoreMock()
	store.GetBalanceByGroupIdFn = func(string) ([]types.Balance, error) {
		stages = append(stages, "balances")
		return nil, wantErr
	}
	groupStore := groupStoreMock()
	groupStore.GetGroupCurrencyFn = func(string) (string, error) {
		stages = append(stages, "currency")
		return "CAD", nil
	}
	handler := NewHandler(store, userStoreMock(), groupStore, expenseControllerMock())
	context, recorder := newReadContext(gin.Params{{Key: "groupId", Value: uuid.NewString()}})

	handler.handleGetUnsettledBalance(context)

	requireInternalError(t, recorder)
	require.Equal(t, []string{"balances"}, stages)
}

func newReadContext(params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.ReleaseMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = params
	return context, recorder
}

func requireInternalError(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	var response utils.APIErrorResponse
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	require.Equal(t, "internal_error", response.Code)
}
