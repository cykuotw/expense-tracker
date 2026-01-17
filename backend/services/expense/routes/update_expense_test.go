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
			expectStatusCode: http.StatusBadRequest,
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
			expectStatusCode: http.StatusForbidden,
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
				validation.ValidateExpenseExist(store),
				extractors.ExtractExpenseUpdatePayload(),
				validation.ValidateGroupUserPairExist(groupStore),
				handler.handleUpdateExpense,
			)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}
