package expense

import (
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
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
	store := &mockGetExpenseListStore{}
	userStore := &mockGetExpenseListUserStore{}
	groupStore := &mockGetExpenseListGroupStore{}
	controller := &mockExpenseController{}

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		groupID          string
		page             int
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
			name:             "invalid page",
			groupID:          mockGroupID.String(),
			page:             mockTotalPage + 1,
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
			expectResponse:   nil,
		},
		{
			name:             "invalid group id",
			groupID:          uuid.NewString(),
			page:             0,
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
			expectResponse:   nil,
		},
		{
			name:             "invalid empty group id",
			groupID:          uuid.Nil.String(),
			page:             0,
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
			expectResponse:   nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			url := "/expense_list/" + test.groupID + "/" + strconv.Itoa(test.page)
			if test.page == -1 {
				url = "/expense_list/" + test.groupID

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
			router.GET("/expense_list/:groupId", handler.handleGetExpenseList)
			router.GET("/expense_list/:groupId/:page", handler.handleGetExpenseList)

			router.ServeHTTP(rr, req)

			var rsp []types.ExpenseResponseBrief
			if !test.expectFail {
				err = json.NewDecoder(rr.Body).Decode(&rsp)
				if err != nil {
					t.Fatal()
				}
			}

			assert.Equal(t, test.expectStatusCode, rr.Code)
			if !test.expectFail {
				if assert.Equal(t, len(test.expectResponse), len(rsp)) {
					for i, r := range rsp {
						assert.Equal(t, test.expectResponse[i].ExpenseID, r.ExpenseID)
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

type mockGetExpenseListStore struct {
	mockExpenseStore
}

func (s *mockGetExpenseListStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	if page > int64(mockTotalPage) {
		return nil, types.ErrNoRemainingExpenses
	}

	expense := []*types.Expense{
		{
			ID: mockExpenseIDs[0],
		},
		{
			ID: mockExpenseIDs[1],
		},
		{
			ID: mockExpenseIDs[2],
		},
	}
	return expense, nil
}

type mockGetExpenseListGroupStore struct {
	mockGroupStore
}

func (m *mockGetExpenseListGroupStore) CheckGroupExistById(id string) (bool, error) {
	if id == mockGroupID.String() {
		return true, nil
	}
	return false, nil
}
func (m *mockGetExpenseListGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockGetExpenseListUserStore struct {
	mockUserStore
}
