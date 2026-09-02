package expense

import (
	"bytes"
	"encoding/json"
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
	"github.com/stretchr/testify/assert"
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
			context.Set("expense", &types.Expense{ID: mockExpenseID, GroupID: mockGroupID, OccurredOn: "2026-08-30"})
			context.Set("expensePayload", types.ExpenseUpdatePayload{
				GroupID: mockGroupID, PayByUserId: mockCreatorID.String(), ExpenseTypeID: mockExpenseTypeID, OccurredOn: test.requested,
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
