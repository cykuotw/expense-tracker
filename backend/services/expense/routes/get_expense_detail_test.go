package expense

import (
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

func TestRouteGetExpenseDetail(t *testing.T) {
	store := getExpenseDetailStoreMock()
	userStore := getExpenseDetailUserStoreMock()
	groupStore := getExpenseDetailGroupStoreMock()
	controller := expenseControllerMock()

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		expenseID        string
		groupID          string
		expectFail       bool
		expectStatusCode int
		expectResponse   types.ExpenseResponse
	}

	subtests := []testcase{
		{
			name:             "valid",
			expenseID:        mockExpenseID.String(),
			groupID:          mockGroupID.String(),
			expectFail:       false,
			expectStatusCode: http.StatusOK,
			expectResponse: types.ExpenseResponse{
				ID: mockExpenseID,
				Items: []types.ItemResponse{
					{
						ItemID: mockItemIDs[0],
					},
					{
						ItemID: mockItemIDs[1],
					},
					{
						ItemID: mockItemIDs[2],
					},
				},
			},
		},
		{
			name:             "invalid expense id",
			expenseID:        uuid.NewString(),
			groupID:          mockGroupID.String(),
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
			expectResponse:   types.ExpenseResponse{},
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/expense/"+test.expenseID, nil)
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
			router.GET(
				"/expense/:expenseId",
				extractors.ExtractUserIdFromJWT(),
				validation.ValidateExpenseExist(store),
				extractors.ExtractExpenseFromStore(store),
				validation.ValidateGroupUserPairExist(groupStore),
				handler.handleGetExpenseDetail,
			)

			router.ServeHTTP(rr, req)

			var rsp types.ExpenseResponse
			err = json.NewDecoder(rr.Body).Decode(&rsp)
			if err != nil {
				t.Fatal()
			}

			assert.Equal(t, test.expectStatusCode, rr.Code)
			assert.Equal(t, test.expectResponse.ID, rsp.ID)
			if assert.Equal(t, len(test.expectResponse.Items), len(rsp.Items)) {
				for i, it := range rsp.Items {
					assert.Equal(t, test.expectResponse.Items[i].ItemID, it.ItemID)
				}
			}
		})
	}
}
