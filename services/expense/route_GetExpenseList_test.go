package expense

import (
	"encoding/json"
	"expense-tracker/config"
	"expense-tracker/services/auth"
	"expense-tracker/types"
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
			name:             "invalid group id",
			groupID:          uuid.NewString(),
			page:             0,
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
			expectResponse:   nil,
		},
		{
			name:             "invalid empty group id",
			groupID:          uuid.Nil.String(),
			page:             0,
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
			expectResponse:   nil,
		},
		{
			name:             "invalid page",
			groupID:          uuid.Nil.String(),
			page:             mockTotalPage + 1,
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
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

type mockGetExpenseListStore struct{}

func (s *mockGetExpenseListStore) CreateExpense(expense types.Expense) error {
	return nil
}
func (s *mockGetExpenseListStore) CreateItem(item types.Item) error {
	return nil
}
func (s *mockGetExpenseListStore) CreateLedger(ledger types.Ledger) error {
	return nil
}
func (s *mockGetExpenseListStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	return nil, nil
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
func (s *mockGetExpenseListStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return nil, nil
}
func (s *mockGetExpenseListStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockGetExpenseListStore) GetLedgerUnsettledFromGroup(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockGetExpenseListStore) UpdateExpense(expense types.Expense) error {
	return nil
}
func (s *mockGetExpenseListStore) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}
func (s *mockGetExpenseListStore) UpdateItem(item types.Item) error {
	return nil
}
func (s *mockGetExpenseListStore) UpdateLedger(ledger types.Ledger) error {
	return nil
}

type mockGetExpenseListGroupStore struct{}

func (m *mockGetExpenseListGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGetExpenseListGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGetExpenseListGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if groupID != mockGroupID.String() {
		return nil, types.ErrGroupNotExist
	}
	if userID != mockUserID.String() {
		return nil, types.ErrUserNotExist
	}
	return nil, nil
}
func (m *mockGetExpenseListGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockGetExpenseListGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockGetExpenseListGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGetExpenseListGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGetExpenseListGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}

type mockGetExpenseListUserStore struct{}

func (m *mockGetExpenseListUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetExpenseListUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetExpenseListUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetExpenseListUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockGetExpenseListUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockGetExpenseListUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
