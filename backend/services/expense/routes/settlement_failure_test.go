package expense

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHandleSettleExpenseFailureStagesRollBack(t *testing.T) {
	stageErr := errors.New("injected settlement failure")
	tests := []struct {
		name         string
		failureStage string
		wantStatus   int
		wantCommit   bool
	}{
		{name: "transaction boundary", failureStage: "transaction boundary", wantStatus: http.StatusInternalServerError},
		{name: "inactive group", failureStage: "group lock", wantStatus: http.StatusNotFound},
		{name: "membership changed", failureStage: "membership", wantStatus: http.StatusNotFound},
		{name: "expense update", failureStage: "expense update", wantStatus: http.StatusInternalServerError},
		{name: "ledger read", failureStage: "ledger read", wantStatus: http.StatusInternalServerError},
		{name: "balance outdate", failureStage: "balance outdate", wantStatus: http.StatusInternalServerError},
		{name: "balance creation", failureStage: "balance creation", wantStatus: http.StatusInternalServerError},
		{name: "relationship conflict", failureStage: "relationship conflict", wantStatus: http.StatusConflict},
		{name: "success", wantStatus: http.StatusCreated, wantCommit: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := struct {
				expensesSettled     bool
				oldBalancesOutdated bool
				newBalancesCreated  bool
				relationships       bool
			}{}
			transactionCalled := false
			store := settleExpenseStoreMock()
			store.RunInTransactionFn = func(callback func(types.ExpenseTransactionStore) error) error {
				transactionCalled = true
				if test.failureStage == "transaction boundary" {
					return stageErr
				}
				before := state
				err := callback(store)
				if err != nil {
					state = before
				}
				return err
			}
			store.LockGroupCurrencyFn = func(string) (string, error) {
				if test.failureStage == "group lock" {
					return "", types.ErrGroupNotExist
				}
				return "CAD", nil
			}
			store.CheckGroupParticipantsFn = func(string, []uuid.UUID) error {
				if test.failureStage == "membership" {
					return types.ErrGroupParticipantNotAllowed
				}
				return nil
			}
			store.UpdateExpenseSettleInGroupFn = func(string) error {
				if test.failureStage == "expense update" {
					return stageErr
				}
				state.expensesSettled = true
				return nil
			}
			store.GetLedgerUnsettledFromGroupFn = func(string) ([]*types.Ledger, error) {
				if test.failureStage == "ledger read" {
					return nil, stageErr
				}
				return nil, nil
			}
			store.OutdateBalanceByGroupIdFn = func(string) error {
				if test.failureStage == "balance outdate" {
					return stageErr
				}
				state.oldBalancesOutdated = true
				return nil
			}
			store.CreateBalancesFn = func(string, []*types.Balance) error {
				if test.failureStage == "balance creation" {
					return stageErr
				}
				state.newBalancesCreated = true
				return nil
			}
			store.CreateBalanceLedgerFn = func([]uuid.UUID, []uuid.UUID) error {
				if test.failureStage == "relationship conflict" {
					return types.ErrBalanceLedgerConflict
				}
				state.relationships = true
				return nil
			}

			response := runSettleExpenseHandler(t, store)

			assert.Equal(t, test.wantStatus, response.Code)
			assert.Equal(t, test.wantCommit, state.expensesSettled)
			assert.Equal(t, test.wantCommit, state.oldBalancesOutdated)
			assert.Equal(t, test.wantCommit, state.newBalancesCreated)
			assert.Equal(t, test.wantCommit, state.relationships)
			assert.True(t, transactionCalled)
		})
	}
}

func TestHandleSettleBalanceFailureStagesRollBack(t *testing.T) {
	stageErr := errors.New("injected balance settlement failure")
	tests := []struct {
		name                string
		failureStage        string
		allSettled          bool
		wantStatus          int
		wantBalanceSettled  bool
		wantExpensesSettled bool
	}{
		{name: "transaction boundary", failureStage: "transaction boundary", wantStatus: http.StatusInternalServerError},
		{name: "inactive group", failureStage: "group lock", wantStatus: http.StatusNotFound},
		{name: "membership changed", failureStage: "membership", wantStatus: http.StatusNotFound},
		{name: "balance not found", failureStage: "balance not found", wantStatus: http.StatusNotFound},
		{name: "not a balance party", failureStage: "not a balance party", wantStatus: http.StatusForbidden},
		{name: "balance update", failureStage: "balance update", wantStatus: http.StatusInternalServerError},
		{name: "final status check", failureStage: "final status check", wantStatus: http.StatusInternalServerError},
		{name: "more balances remain", wantStatus: http.StatusCreated, wantBalanceSettled: true},
		{name: "final expense update", failureStage: "final expense update", allSettled: true, wantStatus: http.StatusInternalServerError},
		{name: "final balance success", allSettled: true, wantStatus: http.StatusCreated, wantBalanceSettled: true, wantExpensesSettled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			balanceSettled := false
			expensesSettled := false
			transactionCalled := false
			store := settleExpenseStoreMock()
			store.RunInTransactionFn = func(callback func(types.ExpenseTransactionStore) error) error {
				transactionCalled = true
				if test.failureStage == "transaction boundary" {
					return stageErr
				}
				beforeBalance := balanceSettled
				beforeExpenses := expensesSettled
				err := callback(store)
				if err != nil {
					balanceSettled = beforeBalance
					expensesSettled = beforeExpenses
				}
				return err
			}
			store.LockGroupCurrencyFn = func(string) (string, error) {
				if test.failureStage == "group lock" {
					return "", types.ErrGroupNotExist
				}
				return "CAD", nil
			}
			store.CheckGroupParticipantsFn = func(string, []uuid.UUID) error {
				if test.failureStage == "membership" {
					return types.ErrGroupParticipantNotAllowed
				}
				return nil
			}
			store.SettleBalanceByBalanceIDFn = func(string, string, uuid.UUID) error {
				if test.failureStage == "balance not found" {
					return types.ErrBalanceNotExist
				}
				if test.failureStage == "not a balance party" {
					return types.ErrUserNotPermitted
				}
				if test.failureStage == "balance update" {
					return stageErr
				}
				balanceSettled = true
				return nil
			}
			store.CheckGroupBalanceAllSettledFn = func(string) (bool, error) {
				if test.failureStage == "final status check" {
					return false, stageErr
				}
				return test.allSettled, nil
			}
			store.UpdateExpenseSettleInGroupFn = func(string) error {
				if test.failureStage == "final expense update" {
					return stageErr
				}
				expensesSettled = true
				return nil
			}

			response := runSettleBalanceHandler(t, store)

			assert.Equal(t, test.wantStatus, response.Code)
			assert.Equal(t, test.wantBalanceSettled, balanceSettled)
			assert.Equal(t, test.wantExpensesSettled, expensesSettled)
			assert.True(t, transactionCalled)
		})
	}
}

func TestHandleDeleteExpenseFailureStagesRollBack(t *testing.T) {
	stageErr := errors.New("injected deletion failure")
	tests := []struct {
		name         string
		failureStage string
		wantStatus   int
		wantCommit   bool
	}{
		{name: "transaction boundary", failureStage: "transaction boundary", wantStatus: http.StatusInternalServerError},
		{name: "inactive group", failureStage: "group lock", wantStatus: http.StatusNotFound},
		{name: "membership changed", failureStage: "membership", wantStatus: http.StatusNotFound},
		{name: "expense deletion", failureStage: "expense deletion", wantStatus: http.StatusInternalServerError},
		{name: "ledger read", failureStage: "ledger read", wantStatus: http.StatusInternalServerError},
		{name: "balance outdate", failureStage: "balance outdate", wantStatus: http.StatusInternalServerError},
		{name: "balance creation", failureStage: "balance creation", wantStatus: http.StatusInternalServerError},
		{name: "relationship conflict", failureStage: "relationship conflict", wantStatus: http.StatusConflict},
		{name: "success", wantStatus: http.StatusOK, wantCommit: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := struct {
				expenseDeleted      bool
				oldBalancesOutdated bool
				newBalancesCreated  bool
				relationships       bool
			}{}
			transactionCalled := false
			store := expenseStoreMock()
			store.RunInTransactionFn = func(callback func(types.ExpenseTransactionStore) error) error {
				transactionCalled = true
				if test.failureStage == "transaction boundary" {
					return stageErr
				}
				before := state
				err := callback(store)
				if err != nil {
					state = before
				}
				return err
			}
			store.LockGroupCurrencyFn = func(string) (string, error) {
				if test.failureStage == "group lock" {
					return "", types.ErrGroupNotExist
				}
				return "CAD", nil
			}
			store.CheckGroupParticipantsFn = func(string, []uuid.UUID) error {
				if test.failureStage == "membership" {
					return types.ErrGroupParticipantNotAllowed
				}
				return nil
			}
			store.DeleteExpenseFn = func(types.Expense) error {
				if test.failureStage == "expense deletion" {
					return stageErr
				}
				state.expenseDeleted = true
				return nil
			}
			store.GetLedgerUnsettledFromGroupFn = func(string) ([]*types.Ledger, error) {
				if test.failureStage == "ledger read" {
					return nil, stageErr
				}
				return nil, nil
			}
			store.OutdateBalanceByGroupIdFn = func(string) error {
				if test.failureStage == "balance outdate" {
					return stageErr
				}
				state.oldBalancesOutdated = true
				return nil
			}
			store.CreateBalancesFn = func(string, []*types.Balance) error {
				if test.failureStage == "balance creation" {
					return stageErr
				}
				state.newBalancesCreated = true
				return nil
			}
			store.CreateBalanceLedgerFn = func([]uuid.UUID, []uuid.UUID) error {
				if test.failureStage == "relationship conflict" {
					return types.ErrBalanceLedgerConflict
				}
				state.relationships = true
				return nil
			}

			response := runDeleteExpenseHandler(t, store)

			assert.Equal(t, test.wantStatus, response.Code)
			assert.Equal(t, test.wantCommit, state.expenseDeleted)
			assert.Equal(t, test.wantCommit, state.oldBalancesOutdated)
			assert.Equal(t, test.wantCommit, state.newBalancesCreated)
			assert.Equal(t, test.wantCommit, state.relationships)
			assert.True(t, transactionCalled)
		})
	}
}

func runSettleExpenseHandler(t *testing.T, store *mockExpenseStore) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Params = gin.Params{{Key: "groupId", Value: mockGroupID.String()}}
	context.Set("userID", mockUserID.String())
	context.Request = httptest.NewRequest(http.MethodPut, "/settle_expense/"+mockGroupID.String(), nil)

	handler := NewHandler(store, nil, nil, expenseControllerMock())
	handler.handleSettleExpense(context)
	return response
}

func runSettleBalanceHandler(t *testing.T, store *mockExpenseStore) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Params = gin.Params{
		{Key: "groupId", Value: mockGroupID.String()},
		{Key: "balanceId", Value: uuid.NewString()},
	}
	context.Set("userID", mockUserID.String())
	context.Request = httptest.NewRequest(http.MethodPost, "/settle_balance", nil)

	handler := NewHandler(store, nil, nil, expenseControllerMock())
	handler.handleSettleBalance(context)
	return response
}

func runDeleteExpenseHandler(t *testing.T, store *mockExpenseStore) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("expense", &types.Expense{ID: mockExpenseID, GroupID: mockGroupID})
	context.Set("userID", mockUserID.String())
	context.Request = httptest.NewRequest(http.MethodDelete, "/expense/"+mockExpenseID.String(), nil)

	handler := NewHandler(store, nil, nil, expenseControllerMock())
	handler.handleDeleteExpense(context)
	return response
}
