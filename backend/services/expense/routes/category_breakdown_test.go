package expense

import (
	"errors"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func categoryRows() []types.ExpenseCategoryAmount {
	return []types.ExpenseCategoryAmount{{ExpenseTypeID: mockExpenseTypeID, Amount: decimal.RequireFromString("200"), Note: "Groceries"}, {ExpenseTypeID: uuid.New(), Amount: decimal.RequireFromString("100"), Note: "Trolley"}}
}

func TestCategoryBreakdownValidation(t *testing.T) {
	expense := types.Expense{ExpenseTypeID: mockExpenseTypeID, Total: decimal.NewFromInt(300)}
	require.NoError(t, validateCategoryBreakdown(expense, nil, 2))
	require.NoError(t, validateCategoryBreakdown(expense, categoryRows(), 2))
	require.NoError(t, validateCategoryBreakdown(expense, categoryRows(), 0))
	for _, tc := range []struct {
		name   string
		change func([]types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount
	}{
		{"one row", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount { return rows[:1] }},
		{"duplicate", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[1].ExpenseTypeID = rows[0].ExpenseTypeID
			return rows
		}},
		{"missing type", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[1].ExpenseTypeID = uuid.Nil
			return rows
		}},
		{"negative", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[1].Amount = decimal.NewFromInt(-1)
			return rows
		}},
		{"zero", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[1].Amount = decimal.Zero
			return rows
		}},
		{"too much", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[1].Amount = decimal.NewFromInt(101)
			return rows
		}},
		{"too little", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[1].Amount = decimal.NewFromInt(99)
			return rows
		}},
		{"precision", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[0].Amount = decimal.RequireFromString("199.999")
			rows[1].Amount = decimal.RequireFromString("100.001")
			return rows
		}},
		{"long note", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[1].Note = strings.Repeat("字", 201)
			return rows
		}},
		{"parent mismatch", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			rows[0].ExpenseTypeID = uuid.New()
			return rows
		}},
		{"row limit", func(rows []types.ExpenseCategoryAmount) []types.ExpenseCategoryAmount {
			return make([]types.ExpenseCategoryAmount, 21)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, validateCategoryBreakdown(expense, tc.change(categoryRows()), 2), types.ErrInvalidCategoryBreakdown)
		})
	}
	twenty := make([]types.ExpenseCategoryAmount, 20)
	for i := range twenty {
		twenty[i] = types.ExpenseCategoryAmount{ExpenseTypeID: uuid.New(), Amount: decimal.NewFromInt(15)}
	}
	twenty[0].ExpenseTypeID = mockExpenseTypeID
	require.NoError(t, validateCategoryBreakdown(expense, twenty, 2))
	thirds := categoryRows()
	thirds[0].Amount = decimal.RequireFromString("199.999")
	thirds[1].Amount = decimal.RequireFromString("100.001")
	require.NoError(t, validateCategoryBreakdown(expense, thirds, 3))
}

func TestCategoryBreakdownCreateUsesTransactionAndFingerprint(t *testing.T) {
	payload := validCreateExpensePayload()
	payload.Total = decimal.NewFromInt(300)
	payload.SubTotal = decimal.Zero
	payload.TaxFeeTip = decimal.Zero
	payload.Items = nil
	payload.Allocation = exactAllocation(mockCreatorID, "300")
	payload.CategoryBreakdown = categoryRows()
	root, tx := createExpenseStoreMock(), createExpenseStoreMock()
	root.RunInTransactionFn = func(callback func(types.ExpenseTransactionStore) error) error { return callback(tx) }
	root.ReplaceExpenseCategoryBreakdownFn = func(uuid.UUID, []types.ExpenseCategoryAmount) error { t.Fatal("write escaped transaction"); return nil }
	writes := 0
	tx.ReplaceExpenseCategoryBreakdownFn = func(id uuid.UUID, rows []types.ExpenseCategoryAmount) error {
		writes++
		require.NotEqual(t, uuid.Nil, id)
		require.Equal(t, payload.CategoryBreakdown, rows)
		return nil
	}
	response := runCreateExpenseHandler(t, root, payload)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.Equal(t, 1, writes)
	expense := types.Expense{ExpenseTypeID: mockExpenseTypeID, CategoryBreakdown: payload.CategoryBreakdown}
	first, err := expenseCreateFingerprint(expense, nil, nil, nil)
	require.NoError(t, err)
	expense.CategoryBreakdown[1].Note = "Different use"
	second, err := expenseCreateFingerprint(expense, nil, nil, nil)
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	tx.ReplaceExpenseCategoryBreakdownFn = func(uuid.UUID, []types.ExpenseCategoryAmount) error { return errors.New("child insert failed") }
	require.Equal(t, http.StatusInternalServerError, runCreateExpenseHandler(t, root, payload).Code)
}

func TestCategoryBreakdownUpdateRequiresExplicitPreservationOrRemoval(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "omitted", true: "explicit removal"}[explicit], func(t *testing.T) {
			store := updateExpenseDetailStoreMock()
			store.GetCategoryBreakdownsByExpenseIDsFn = func(ids []string) (map[string][]types.ExpenseCategoryAmount, error) {
				return map[string][]types.ExpenseCategoryAmount{mockExpenseID.String(): categoryRows()}, nil
			}
			writes := 0
			store.ReplaceExpenseCategoryBreakdownFn = func(_ uuid.UUID, rows []types.ExpenseCategoryAmount) error {
				writes++
				require.Empty(t, rows)
				return nil
			}
			payload := types.ExpenseUpdatePayload{GroupID: mockGroupID, PayByUserId: mockCreatorID.String(), ExpenseTypeID: mockExpenseTypeID, Total: decimal.NewFromInt(1), Currency: "CAD", Allocation: exactAllocation(mockUserID, "1")}
			if explicit {
				payload.CategoryBreakdown = []types.ExpenseCategoryAmount{}
			}
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Set("userID", mockUserID.String())
			context.Set("expensePayload", payload)
			context.Set("expense", &types.Expense{ID: mockExpenseID, GroupID: mockGroupID, Currency: "CAD"})
			context.Request = httptest.NewRequest(http.MethodPut, "/expense/"+mockExpenseID.String(), nil)
			NewHandler(store, nil, nil, expenseControllerMock()).handleUpdateExpense(context)
			if explicit {
				require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
				require.Equal(t, 1, writes)
			} else {
				require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
				require.Contains(t, response.Body.String(), "category_breakdown_required")
				require.Zero(t, writes)
			}
		})
	}
}

func TestCategoryBreakdownRetryDoesNotDuplicatePurchase(t *testing.T) {
	payload := validCreateExpensePayload()
	payload.Total = decimal.NewFromInt(300)
	payload.SubTotal = decimal.Zero
	payload.TaxFeeTip = decimal.Zero
	payload.Items = nil
	payload.Allocation = exactAllocation(mockCreatorID, "300")
	payload.CategoryBreakdown = categoryRows()
	store := createExpenseStoreMock()
	var claimed *types.ExpenseCreateIdempotency
	store.ClaimExpenseCreateIdempotencyFn = func(record types.ExpenseCreateIdempotency) (types.ExpenseCreateIdempotency, bool, error) {
		if claimed != nil {
			return *claimed, false, nil
		}
		claimed = &record
		return record, true, nil
	}
	writes := 0
	store.CreateExpenseFn = func(types.Expense) error { writes++; return nil }
	key := uuid.NewString()
	run := func() *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(response)
		context.Set("userID", mockCreatorID.String())
		context.Set("expensePayload", payload)
		context.Request = httptest.NewRequest(http.MethodPost, "/create_expense", nil)
		context.Request.Header.Set("Idempotency-Key", key)
		NewHandler(store, nil, nil, expenseControllerMock()).handleCreateExpense(context)
		return response
	}
	first, replay := run(), run()
	require.Equal(t, http.StatusCreated, first.Code)
	require.Equal(t, first.Body.String(), replay.Body.String())
	require.Equal(t, 1, writes)
	payload.CategoryBreakdown[0].Amount = decimal.NewFromInt(190)
	payload.CategoryBreakdown[1].Amount = decimal.NewFromInt(110)
	require.Equal(t, http.StatusConflict, run().Code)
	require.Equal(t, 1, writes)
}
