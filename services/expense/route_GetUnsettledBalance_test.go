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

func TestRouteGetUnsettledBalance(t *testing.T) {
	store := &mockGetUnsettledBalanceStore{}
	userStore := &mockGetUnsettledBalanceUserStore{}
	groupStore := &mockGetUnsettledBalanceGroupStore{}
	controller := &mockGetUnsettledBalanceController{}

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		groupID          string
		expectFail       bool
		expectStatusCode int
		expectResponse   types.BalanceResponse
	}

	subtests := []testcase{
		{
			name:             "valid",
			groupID:          mockGroupID.String(),
			expectFail:       false,
			expectStatusCode: http.StatusOK,
			expectResponse: types.BalanceResponse{
				Currency: mockCurrency,
				Balances: []types.BalanceRsp{
					{
						SenderUserID:   mockSenderIDs[0],
						ReceiverUserID: mockReceiverIDs[0],
					},
					{
						SenderUserID:   mockSenderIDs[1],
						ReceiverUserID: mockReceiverIDs[1],
					},
					{
						SenderUserID:   mockSenderIDs[2],
						ReceiverUserID: mockReceiverIDs[2],
					},
				},
			},
		},
		{
			name:             "invalid group id",
			groupID:          uuid.NewString(),
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
			expectResponse:   types.BalanceResponse{},
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/balance/"+test.groupID, nil)
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
			router.GET("/balance/:groupId", handler.handleGetUnsettledBalance)

			router.ServeHTTP(rr, req)

			var rsp types.BalanceResponse
			err = json.NewDecoder(rr.Body).Decode(&rsp)
			if err != nil {
				t.Fatal()
			}

			assert.Equal(t, test.expectStatusCode, rr.Code)
			assert.Equal(t, test.expectResponse.Currency, rsp.Currency)
			if assert.Equal(t, len(test.expectResponse.Balances), len(rsp.Balances)) {
				for i, b := range rsp.Balances {
					assert.Equal(t, test.expectResponse.Balances[i].SenderUserID, b.SenderUserID)
					assert.Equal(t, test.expectResponse.Balances[i].ReceiverUserID, b.ReceiverUserID)
				}
			}
		})
	}
}

var mockUserIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockLedgerIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockSenderIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockReceiverIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockCurrency = "CAD"
var mockLedger = []*types.Ledger{
	{
		ID: mockLedgerIDs[0],
	},
	{
		ID: mockLedgerIDs[1],
	},
	{
		ID: mockLedgerIDs[2],
	},
}
var mockBalance = []*types.Balance{
	{
		SenderUserID:   mockSenderIDs[0],
		ReceiverUserID: mockReceiverIDs[0],
	},
	{
		SenderUserID:   mockSenderIDs[1],
		ReceiverUserID: mockReceiverIDs[1],
	},
	{
		SenderUserID:   mockSenderIDs[2],
		ReceiverUserID: mockReceiverIDs[2],
	},
}

type mockGetUnsettledBalanceStore struct{}

func (s *mockGetUnsettledBalanceStore) CreateExpense(expense types.Expense) error {
	return nil
}
func (s *mockGetUnsettledBalanceStore) CreateItem(item types.Item) error {
	return nil
}
func (s *mockGetUnsettledBalanceStore) CreateLedger(ledger types.Ledger) error {
	return nil
}
func (s *mockGetUnsettledBalanceStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	return nil, nil
}
func (s *mockGetUnsettledBalanceStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	return nil, nil
}
func (s *mockGetUnsettledBalanceStore) GetExpenseType() ([]*types.ExpenseType, error) {
	return nil, nil
}
func (s *mockGetUnsettledBalanceStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return mockItems, nil
}
func (s *mockGetUnsettledBalanceStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockGetUnsettledBalanceStore) GetLedgerUnsettledFromGroup(groupID string) ([]*types.Ledger, error) {
	if groupID != mockGroupID.String() {
		return nil, nil
	}

	return mockLedger, nil
}
func (s *mockGetUnsettledBalanceStore) UpdateExpense(expense types.Expense) error {
	return nil
}
func (s *mockGetUnsettledBalanceStore) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}
func (s *mockGetUnsettledBalanceStore) UpdateItem(item types.Item) error {
	return nil
}
func (s *mockGetUnsettledBalanceStore) UpdateLedger(ledger types.Ledger) error {
	return nil
}
func (m *mockGetUnsettledBalanceStore) CheckExpenseExistByID(id string) (bool, error) {
	return false, nil
}

type mockGetUnsettledBalanceGroupStore struct{}

func (m *mockGetUnsettledBalanceGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGetUnsettledBalanceGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGetUnsettledBalanceGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if groupID != mockGroupID.String() {
		return nil, types.ErrGroupNotExist
	}
	if userID != mockUserID.String() {
		return nil, types.ErrUserNotExist
	}
	return nil, nil
}
func (m *mockGetUnsettledBalanceGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockGetUnsettledBalanceGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockGetUnsettledBalanceGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGetUnsettledBalanceGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGetUnsettledBalanceGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return mockCurrency, nil
}
func (m *mockGetUnsettledBalanceGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockGetUnsettledBalanceGroupStore) CheckGroupExistById(id string) (bool, error) {
	return false, nil
}
func (m *mockGetUnsettledBalanceGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	return false, nil
}

type mockGetUnsettledBalanceUserStore struct{}

func (m *mockGetUnsettledBalanceUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetUnsettledBalanceUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetUnsettledBalanceUserStore) GetUserByID(id string) (*types.User, error) {
	user := &types.User{
		ID:       mockUserID,
		Username: "test user",
	}
	return user, nil
}
func (m *mockGetUnsettledBalanceUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockGetUnsettledBalanceUserStore) GetUsernameByID(userid string) (string, error) {
	return "test", nil
}
func (m *mockGetUnsettledBalanceUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockGetUnsettledBalanceUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockGetUnsettledBalanceUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockGetUnsettledBalanceUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockGetUnsettledBalanceController struct{}

func (c *mockGetUnsettledBalanceController) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	return mockBalance
}
