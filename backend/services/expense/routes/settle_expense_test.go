package expense

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/services/middleware/validation"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRouteSettleExpense(t *testing.T) {
	store := settleExpenseStoreMock()
	userStore := settleExpenseUserStoreMock()
	groupStore := settleExpenseGroupStoreMock()
	controller := expenseControllerMock()

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		groupID          string
		expectFail       bool
		expectStatusCode int
	}

	subtests := []testcase{
		{
			name:             "valid",
			groupID:          mockGroupID.String(),
			expectFail:       false,
			expectStatusCode: http.StatusCreated,
		},
		{
			name:             "invalid group id",
			groupID:          uuid.New().String(),
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPut, "/settle_expense/"+test.groupID, nil)
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
				"/settle_expense/:groupId",
				extractors.ExtractUserIdFromJWT(),
				validation.ValidateGroupUserPairExist(groupStore),
				handler.handleSettleExpense,
			)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}
