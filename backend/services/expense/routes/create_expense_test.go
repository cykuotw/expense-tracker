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
)

func TestRouteCreateExpense(t *testing.T) {
	store := createExpenseStoreMock()
	userStore := createExpenseUserStoreMock()
	groupStore := createExpenseGroupStoreMock()
	controller := expenseControllerMock()

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		payload          types.ExpensePayload
		expectFail       bool
		expectStatusCode int
	}

	subtests := []testcase{
		{
			name: "valid",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: mockCreatorID.String(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       false,
			expectStatusCode: http.StatusCreated,
		},
		{
			name: "invalid user id",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: uuid.NewString(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       true,
			expectStatusCode: http.StatusNotFound,
		},
		{
			name: "invalid group id",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        uuid.NewString(),
				CreateByUserID: mockCreatorID.String(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       true,
			expectStatusCode: http.StatusNotFound,
		},
		{
			name: "invalid group id",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: mockUserID.String(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       true,
			expectStatusCode: http.StatusNotFound,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			marshalled, _ := json.Marshal(test.payload)
			req, err := http.NewRequest(http.MethodPost, "/create_expense", bytes.NewBuffer(marshalled))
			if err != nil {
				t.Fatal(err)
			}

			jwt, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), uuid.MustParse(test.payload.CreateByUserID))
			if err != nil {
				t.Fatal(err)
			}
			req.Header = map[string][]string{
				"Authorization":   {"Bearer " + jwt},
				"Idempotency-Key": {uuid.NewString()},
			}

			rr := httptest.NewRecorder()
			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.POST(
				"/create_expense",
				extractors.ExtractUserIdFromJWT(),
				extractors.ExtractExpensePayload(),
				validation.ValidateGroupUserPairExist(groupStore),
				handler.handleCreateExpense,
			)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}

func TestHandleCreateExpenseUsesOneTransactionBoundStore(t *testing.T) {
	rootStore := createExpenseStoreMock()
	transactionStore := createExpenseStoreMock()
	transactionCalls := 0
	stages := []string{}

	rootStore.RunInTransactionFn = func(callback func(types.ExpenseStore) error) error {
		transactionCalls++
		return callback(transactionStore)
	}
	transactionStore.CreateExpenseFn = func(types.Expense) error {
		stages = append(stages, "expense")
		return nil
	}
	transactionStore.CreateItemFn = func(types.Item) error {
		stages = append(stages, "item")
		return nil
	}
	transactionStore.CreateLedgerFn = func(types.Ledger) error {
		stages = append(stages, "ledger")
		return nil
	}
	transactionStore.GetLedgerUnsettledFromGroupFn = func(string) ([]*types.Ledger, error) {
		stages = append(stages, "ledger read")
		return nil, nil
	}
	transactionStore.OutdateBalanceByGroupIdFn = func(string) error {
		stages = append(stages, "balance outdate")
		return nil
	}
	transactionStore.CreateBalancesFn = func(string, []*types.Balance) error {
		stages = append(stages, "balance create")
		return nil
	}
	transactionStore.CreateBalanceLedgerFn = func([]uuid.UUID, []uuid.UUID) error {
		stages = append(stages, "balance-ledger create")
		return nil
	}

	response := runCreateExpenseHandler(t, rootStore, validCreateExpensePayload())

	assert.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, 1, transactionCalls)
	assert.Equal(t, []string{
		"expense",
		"item",
		"ledger",
		"ledger read",
		"balance outdate",
		"balance create",
		"balance-ledger create",
	}, stages)
}

func TestHandleCreateExpenseRejectsInvalidLedgerBeforeTransaction(t *testing.T) {
	store := createExpenseStoreMock()
	transactionCalls := 0
	store.RunInTransactionFn = func(callback func(types.ExpenseStore) error) error {
		transactionCalls++
		return callback(store)
	}
	payload := validCreateExpensePayload()
	payload.Ledgers[0].LenderUserID = "invalid-uuid"

	response := runCreateExpenseHandler(t, store, payload)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Zero(t, transactionCalls)
}

func TestHandleCreateExpenseRequiresIdempotencyKey(t *testing.T) {
	store := createExpenseStoreMock()
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("userID", mockCreatorID.String())
	context.Set("expensePayload", validCreateExpensePayload())
	context.Request = httptest.NewRequest(http.MethodPost, "/create_expense", nil)

	NewHandler(store, nil, nil, expenseControllerMock()).handleCreateExpense(context)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"code":"invalid_idempotency_key"`)
}

func TestHandleCreateExpenseReplaysMatchingKey(t *testing.T) {
	store := createExpenseStoreMock()
	existingID := uuid.New()
	store.ClaimExpenseCreateIdempotencyFn = func(record types.ExpenseCreateIdempotency) (types.ExpenseCreateIdempotency, bool, error) {
		return types.ExpenseCreateIdempotency{ExpenseID: existingID, RequestFingerprint: record.RequestFingerprint}, false, nil
	}

	response := runCreateExpenseHandler(t, store, validCreateExpensePayload())

	assert.Equal(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), existingID.String())
}

func TestHandleCreateExpenseRejectsKeyReusedForDifferentPayload(t *testing.T) {
	store := createExpenseStoreMock()
	store.ClaimExpenseCreateIdempotencyFn = func(record types.ExpenseCreateIdempotency) (types.ExpenseCreateIdempotency, bool, error) {
		return types.ExpenseCreateIdempotency{ExpenseID: uuid.New(), RequestFingerprint: []byte("different")}, false, nil
	}

	response := runCreateExpenseHandler(t, store, validCreateExpensePayload())

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), `"code":"idempotency_key_conflict"`)
}

func TestHandleCreateExpensePropagatesTransactionStageErrors(t *testing.T) {
	stageErr := errors.New("stage failed")
	tests := []struct {
		name      string
		configure func(rootStore, transactionStore *mockExpenseStore)
	}{
		{
			name: "transaction boundary",
			configure: func(rootStore, _ *mockExpenseStore) {
				rootStore.RunInTransactionFn = func(func(types.ExpenseStore) error) error {
					return stageErr
				}
			},
		},
		{
			name: "expense insert",
			configure: func(_ *mockExpenseStore, transactionStore *mockExpenseStore) {
				transactionStore.CreateExpenseFn = func(types.Expense) error { return stageErr }
			},
		},
		{
			name: "item insert",
			configure: func(_ *mockExpenseStore, transactionStore *mockExpenseStore) {
				transactionStore.CreateItemFn = func(types.Item) error { return stageErr }
			},
		},
		{
			name: "ledger insert",
			configure: func(_ *mockExpenseStore, transactionStore *mockExpenseStore) {
				transactionStore.CreateLedgerFn = func(types.Ledger) error { return stageErr }
			},
		},
		{
			name: "unsettled ledger read",
			configure: func(_ *mockExpenseStore, transactionStore *mockExpenseStore) {
				transactionStore.GetLedgerUnsettledFromGroupFn = func(string) ([]*types.Ledger, error) {
					return nil, stageErr
				}
			},
		},
		{
			name: "balance outdate",
			configure: func(_ *mockExpenseStore, transactionStore *mockExpenseStore) {
				transactionStore.OutdateBalanceByGroupIdFn = func(string) error { return stageErr }
			},
		},
		{
			name: "balance insert",
			configure: func(_ *mockExpenseStore, transactionStore *mockExpenseStore) {
				transactionStore.CreateBalancesFn = func(string, []*types.Balance) error { return stageErr }
			},
		},
		{
			name: "balance-ledger insert",
			configure: func(_ *mockExpenseStore, transactionStore *mockExpenseStore) {
				transactionStore.CreateBalanceLedgerFn = func([]uuid.UUID, []uuid.UUID) error {
					return stageErr
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rootStore := createExpenseStoreMock()
			transactionStore := createExpenseStoreMock()
			rootStore.RunInTransactionFn = func(callback func(types.ExpenseStore) error) error {
				return callback(transactionStore)
			}
			test.configure(rootStore, transactionStore)

			response := runCreateExpenseHandler(t, rootStore, validCreateExpensePayload())

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	}
}

func validCreateExpensePayload() types.ExpensePayload {
	occurredOn := "2026-08-31"
	return types.ExpensePayload{
		Description:    "test desc",
		GroupID:        mockGroupID.String(),
		CreateByUserID: mockCreatorID.String(),
		PayByUserId:    mockPayerID.String(),
		ExpenseTypeID:  mockExpenseTypeID.String(),
		ProviderName:   "test provider",
		SubTotal:       decimal.NewFromFloat(20.1),
		TaxFeeTip:      decimal.NewFromFloat(2.1),
		Total:          decimal.NewFromFloat(22.2),
		Currency:       "CAD",
		OccurredOn:     &occurredOn,
		Items: []types.ItemPayload{
			{
				ItemName:  "test item",
				Amount:    decimal.NewFromInt(1),
				Unit:      "each",
				UnitPrice: decimal.NewFromFloat(20.1),
			},
		},
		Ledgers: []types.LedgerPayload{
			{
				LenderUserID:   mockPayerID.String(),
				BorrowerUesrID: mockCreatorID.String(),
				Share:          decimal.NewFromFloat(22.2),
			},
		},
	}
}

func TestHandleCreateExpensePersistsOccurredOn(t *testing.T) {
	store := createExpenseStoreMock()
	var created types.Expense
	store.CreateExpenseFn = func(expense types.Expense) error {
		created = expense
		return nil
	}

	response := runCreateExpenseHandler(t, store, validCreateExpensePayload())

	assert.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, "2026-08-31", created.OccurredOn)
}

func TestHandleCreateExpenseRejectsMalformedOccurredOn(t *testing.T) {
	store := createExpenseStoreMock()
	payload := validCreateExpensePayload()
	invalid := "2026-02-29"
	payload.OccurredOn = &invalid

	response := runCreateExpenseHandler(t, store, payload)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"code":"invalid_occurred_on"`)
}

func runCreateExpenseHandler(t *testing.T, store *mockExpenseStore, payload types.ExpensePayload) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("userID", mockCreatorID.String())
	context.Set("expensePayload", payload)
	context.Request = httptest.NewRequest(http.MethodPost, "/create_expense", nil)
	context.Request.Header.Set("Idempotency-Key", uuid.NewString())
	handler := NewHandler(store, nil, nil, expenseControllerMock())

	handler.handleCreateExpense(context)
	return response
}
