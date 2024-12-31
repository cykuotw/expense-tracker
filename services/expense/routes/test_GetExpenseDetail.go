package expense

import (
	"encoding/json"
	"expense-tracker/config"
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRouteGetExpenseDetail(t *testing.T) {
	store := &mockGetExpenseDetailStore{}
	userStore := &mockGetExpenseDetailUserStore{}
	groupStore := &mockGetExpenseDetailGroupStore{}
	controller := &mockExpenseController{}

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
			router.GET("/expense/:expenseId", handler.handleGetExpenseDetail)

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

var mockExpenseID = uuid.New()
var mockItemIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockItems = []*types.Item{
	{
		ID: mockItemIDs[0],
	},
	{
		ID: mockItemIDs[1],
	},
	{
		ID: mockItemIDs[2],
	},
}

type mockGetExpenseDetailStore struct{}

func (s *mockGetExpenseDetailStore) CreateExpense(expense types.Expense) error {
	return nil
}
func (s *mockGetExpenseDetailStore) CreateItem(item types.Item) error {
	return nil
}
func (s *mockGetExpenseDetailStore) CreateLedger(ledger types.Ledger) error {
	return nil
}
func (s *mockGetExpenseDetailStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	expense := &types.Expense{
		ID:      mockExpenseID,
		GroupID: mockGroupID,
	}
	return expense, nil
}
func (s *mockGetExpenseDetailStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	return nil, nil
}
func (s *mockGetExpenseDetailStore) GetExpenseType() ([]*types.ExpenseType, error) {
	return nil, nil
}
func (s *mockGetExpenseDetailStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return mockItems, nil
}
func (s *mockGetExpenseDetailStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockGetExpenseDetailStore) GetLedgerUnsettledFromGroup(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockGetExpenseDetailStore) UpdateExpense(expense types.Expense) error {
	return nil
}
func (s *mockGetExpenseDetailStore) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}
func (s *mockGetExpenseDetailStore) UpdateItem(item types.Item) error {
	return nil
}
func (s *mockGetExpenseDetailStore) UpdateLedger(ledger types.Ledger) error {
	return nil
}
func (m *mockGetExpenseDetailStore) CheckExpenseExistByID(id string) (bool, error) {
	if id == mockExpenseID.String() {
		return true, nil
	}
	return false, nil
}
func (m *mockGetExpenseDetailStore) GetExpenseTypeById(id uuid.UUID) (string, error) {
	return "", nil
}
func (m *mockGetExpenseDetailStore) DeleteExpense(expense types.Expense) error {
	return nil
}

type mockGetExpenseDetailGroupStore struct{}

func (m *mockGetExpenseDetailGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGetExpenseDetailGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGetExpenseDetailGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockGetExpenseDetailGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockGetExpenseDetailGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockGetExpenseDetailGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGetExpenseDetailGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGetExpenseDetailGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockGetExpenseDetailGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockGetExpenseDetailGroupStore) CheckGroupExistById(id string) (bool, error) {
	return false, nil
}
func (m *mockGetExpenseDetailGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockGetExpenseDetailUserStore struct{}

func (m *mockGetExpenseDetailUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetExpenseDetailUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetExpenseDetailUserStore) GetUserByID(id string) (*types.User, error) {
	user := &types.User{
		ID:       mockUserID,
		Username: "test user",
	}
	return user, nil
}
func (m *mockGetExpenseDetailUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockGetExpenseDetailUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockGetExpenseDetailUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockGetExpenseDetailUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockGetExpenseDetailUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockGetExpenseDetailUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}
