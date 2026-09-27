package expense

import (
	"bytes"
	"encoding/json"
	"errors"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/services/middleware/validation"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteUpdateExpenseDetail(t *testing.T) {
	store := updateExpenseDetailStoreMock()
	userStore := updateExpenseDetailUserStoreMock()
	groupStore := updateExpenseDetailGroupStoreMock()
	controller := expenseControllerMock()

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		payload          types.ExpenseUpdatePayload
		expenseID        string
		expectFail       bool
		expectStatusCode int
	}

	subtests := []testcase{
		{
			name: "valid",
			payload: types.ExpenseUpdatePayload{
				GroupID:       mockGroupID,
				PayByUserId:   mockCreatorID.String(),
				ExpenseTypeID: mockExpenseTypeID,
				Total:         decimal.NewFromInt(1),
				Currency:      "CAD",
				Allocation:    exactAllocation(mockUserID, "1"),
			},
			expenseID:        mockExpenseID.String(),
			expectFail:       false,
			expectStatusCode: http.StatusCreated,
		},
		{
			name: "invalid expense id",
			payload: types.ExpenseUpdatePayload{
				GroupID:       mockGroupID,
				PayByUserId:   mockCreatorID.String(),
				ExpenseTypeID: mockExpenseTypeID,
				Total:         decimal.NewFromInt(1),
				Currency:      "CAD",
				Allocation:    exactAllocation(mockUserID, "1"),
			},
			expenseID:        uuid.NewString(),
			expectFail:       true,
			expectStatusCode: http.StatusNotFound,
		},
		{
			name: "invalid group id",
			payload: types.ExpenseUpdatePayload{
				GroupID:       uuid.New(),
				PayByUserId:   mockCreatorID.String(),
				ExpenseTypeID: mockExpenseTypeID,
				Total:         decimal.NewFromInt(1),
				Currency:      "CAD",
				Allocation:    exactAllocation(mockUserID, "1"),
			},
			expenseID:        mockExpenseID.String(),
			expectFail:       true,
			expectStatusCode: http.StatusNotFound,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			marshalled, _ := json.Marshal(test.payload)
			req, err := http.NewRequest(http.MethodPut, "/expense/"+test.expenseID, bytes.NewBuffer(marshalled))
			if err != nil {
				t.Fatal()
			}

			jwt, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), mockUserID)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = map[string][]string{
				"Authorization": {"Bearer " + jwt},
			}

			rr := httptest.NewRecorder()
			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.PUT(
				"/expense/:expenseId",
				extractors.ExtractUserIdFromJWT(),
				extractors.ExtractExpenseFromStore(store),
				validation.ValidateGroupUserPairExist(groupStore),
				extractors.ExtractExpenseUpdatePayload(),
				handler.handleUpdateExpense,
			)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}

func TestHandleUpdateExpenseOccurredOnSemantics(t *testing.T) {
	tests := []struct {
		name           string
		requested      *string
		expected       string
		expectedStatus int
	}{
		{name: "omission preserves current date", expected: "2026-08-30", expectedStatus: http.StatusCreated},
		{name: "explicit date replaces current date", requested: stringPointer("2026-08-31"), expected: "2026-08-31", expectedStatus: http.StatusCreated},
		{name: "invalid date is rejected", requested: stringPointer("2026-02-29"), expectedStatus: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := updateExpenseDetailStoreMock()
			var updated types.Expense
			store.UpdateExpenseFn = func(expense types.Expense) error {
				updated = expense
				return nil
			}
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Set("userID", mockCreatorID.String())
			context.Set("expense", &types.Expense{ID: mockExpenseID, GroupID: mockGroupID, OccurredOn: "2026-08-30", Currency: "CAD"})
			context.Set("expensePayload", types.ExpenseUpdatePayload{
				GroupID: mockGroupID, PayByUserId: mockCreatorID.String(), ExpenseTypeID: mockExpenseTypeID, OccurredOn: test.requested,
				Total: decimal.NewFromInt(1), Currency: "CAD", Allocation: exactAllocation(mockUserID, "1"),
			})

			NewHandler(store, nil, updateExpenseDetailGroupStoreMock(), expenseControllerMock()).handleUpdateExpense(context)

			assert.Equal(t, test.expectedStatus, response.Code)
			if test.expectedStatus == http.StatusCreated {
				assert.Equal(t, test.expected, updated.OccurredOn)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }

func TestHandleUpdateExpenseReconcilesItemsInSubmittedOrder(t *testing.T) {
	store := updateExpenseDetailStoreMock()
	existingID := uuid.New()
	totalOne := decimal.NewFromInt(4)
	totalTwo := decimal.NewFromInt(6)
	stages := make([]string, 0, 5)
	var updated types.Item
	var created types.Item
	var retained []uuid.UUID
	store.ClearItemPositionsFn = func(expenseID uuid.UUID) error {
		assert.Equal(t, mockExpenseID, expenseID)
		stages = append(stages, "clear")
		return nil
	}
	store.UpdateItemFn = func(item types.Item) error {
		stages = append(stages, "update")
		updated = item
		return nil
	}
	store.CreateItemFn = func(item types.Item) error {
		stages = append(stages, "create")
		created = item
		return nil
	}
	store.DeleteItemsNotInFn = func(expenseID uuid.UUID, itemIDs []uuid.UUID) error {
		stages = append(stages, "delete omitted")
		retained = append(retained, itemIDs...)
		return nil
	}
	store.UpdateExpenseFn = func(types.Expense) error {
		stages = append(stages, "expense")
		return nil
	}

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("userID", mockCreatorID.String())
	context.Set("expense", &types.Expense{ID: mockExpenseID, GroupID: mockGroupID, Currency: "CAD"})
	context.Set("expensePayload", types.ExpenseUpdatePayload{
		GroupID: mockGroupID, PayByUserId: mockCreatorID.String(), ExpenseTypeID: mockExpenseTypeID,
		SubTotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "CAD",
		Items: []types.ItemUpdatePayload{
			{ID: existingID, ItemPayload: types.ItemPayload{Description: "Existing", LineTotal: &totalOne}},
			{ItemPayload: types.ItemPayload{Description: "New", LineTotal: &totalTwo}},
		},
		Allocation: exactAllocation(mockUserID, "10"),
	})

	NewHandler(store, nil, updateExpenseDetailGroupStoreMock(), expenseControllerMock()).handleUpdateExpense(context)

	require.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, []string{"clear", "update", "create", "delete omitted", "expense"}, stages[:5])
	assert.Equal(t, int32(0), updated.Position)
	assert.Equal(t, int32(1), created.Position)
	require.Len(t, retained, 2)
	assert.Equal(t, existingID, retained[0])
	assert.Equal(t, created.ID, retained[1])
}

func TestHandleUpdateExpenseStopsAfterItemReconciliationFailure(t *testing.T) {
	store := updateExpenseDetailStoreMock()
	injected := errors.New("delete items failed")
	total := decimal.NewFromInt(1)
	expenseUpdated := false
	store.DeleteItemsNotInFn = func(uuid.UUID, []uuid.UUID) error { return injected }
	store.UpdateExpenseFn = func(types.Expense) error {
		expenseUpdated = true
		return nil
	}
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("userID", mockCreatorID.String())
	context.Set("expense", &types.Expense{ID: mockExpenseID, GroupID: mockGroupID, Currency: "CAD"})
	context.Set("expensePayload", types.ExpenseUpdatePayload{
		GroupID: mockGroupID, PayByUserId: mockCreatorID.String(), ExpenseTypeID: mockExpenseTypeID,
		SubTotal: total, Total: total, Currency: "CAD",
		Items:      []types.ItemUpdatePayload{{ItemPayload: types.ItemPayload{Description: "Item", LineTotal: &total}}},
		Allocation: exactAllocation(mockUserID, "1"),
	})

	NewHandler(store, nil, updateExpenseDetailGroupStoreMock(), expenseControllerMock()).handleUpdateExpense(context)

	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.False(t, expenseUpdated)
}
