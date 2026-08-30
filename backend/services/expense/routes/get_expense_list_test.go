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
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRouteGetExpenseList(t *testing.T) {
	store := getExpenseListStoreMock()
	userStore := getExpenseListUserStoreMock()
	groupStore := getExpenseListGroupStoreMock()
	controller := expenseControllerMock()

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		groupID          string
		page             int
		order            string
		expectFail       bool
		expectStatusCode int
		expectResponse   []types.ExpenseResponseBrief
	}

	subtests := []testcase{
		{
			name:             "valid",
			groupID:          mockGroupID.String(),
			page:             0,
			expectFail:       false,
			expectStatusCode: http.StatusOK,
			expectResponse:   mockGetExpenseListRsp,
		},
		{
			name:             "valid no page num",
			groupID:          mockGroupID.String(),
			page:             -1,
			expectFail:       false,
			expectStatusCode: http.StatusOK,
			expectResponse:   mockGetExpenseListRsp,
		},
		{
			name:             "valid oldest first order",
			groupID:          mockGroupID.String(),
			page:             0,
			order:            "oldest",
			expectFail:       false,
			expectStatusCode: http.StatusOK,
			expectResponse:   mockGetExpenseListRsp,
		},
		{
			name:             "invalid order",
			groupID:          mockGroupID.String(),
			page:             0,
			order:            "random",
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
			expectResponse:   nil,
		},
		{
			name:             "invalid page",
			groupID:          mockGroupID.String(),
			page:             mockTotalPage + 1,
			expectFail:       true,
			expectStatusCode: http.StatusOK,
			expectResponse:   nil,
		},
		{
			name:             "invalid group id",
			groupID:          uuid.NewString(),
			page:             0,
			expectFail:       true,
			expectStatusCode: http.StatusNotFound,
			expectResponse:   nil,
		},
		{
			name:             "invalid empty group id",
			groupID:          uuid.Nil.String(),
			page:             0,
			expectFail:       true,
			expectStatusCode: http.StatusNotFound,
			expectResponse:   nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			url := "/expense_list/" + test.groupID + "/" + strconv.Itoa(test.page)
			if test.page == -1 {
				url = "/expense_list/" + test.groupID

			}
			if test.order != "" {
				url += "?order=" + test.order
			}
			req, err := http.NewRequest(http.MethodGet, url, nil)
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
			router.Use(extractors.ExtractUserIdFromJWT())
			router.GET("/expense_list/:groupId", validation.ValidateGroupUserPairExist(groupStore), handler.handleGetExpenseList)
			router.GET("/expense_list/:groupId/:page", validation.ValidateGroupUserPairExist(groupStore), handler.handleGetExpenseList)

			router.ServeHTTP(rr, req)

			var rsp []types.ExpenseResponseBrief
			if !test.expectFail || test.expectStatusCode == http.StatusOK {
				err = json.NewDecoder(rr.Body).Decode(&rsp)
				if err != nil {
					t.Fatal()
				}
			}

			assert.Equal(t, test.expectStatusCode, rr.Code)
			if !test.expectFail || test.expectStatusCode == http.StatusOK {
				if assert.Equal(t, len(test.expectResponse), len(rsp)) {
					for i, r := range rsp {
						assert.Equal(t, test.expectResponse[i].ExpenseID, r.ExpenseID)
						assert.Equal(t, mockExpenseTypeID, r.ExpenseTypeID)
						assert.Equal(t, "Groceries", r.ExpenseType)
						assert.Equal(t, "Food and Drink", r.ExpenseCategory)
					}
				}
			}
		})
	}
}

var mockTotalPage = 3
var mockExpenseIDs = []uuid.UUID{
	uuid.New(), uuid.New(), uuid.New(),
}
var mockGetExpenseListRsp = []types.ExpenseResponseBrief{
	{
		ExpenseID: mockExpenseIDs[0],
	},
	{
		ExpenseID: mockExpenseIDs[1],
	},
	{
		ExpenseID: mockExpenseIDs[2],
	},
}
