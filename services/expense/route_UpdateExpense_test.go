package expense

import (
	"bytes"
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

func TestRouteUpdateExpenseDetail(t *testing.T) {
	store := &mockUpdateExpenseDetailStore{}
	userStore := &mockUpdateExpenseDetailUserStore{}
	groupStore := &mockUpdateExpenseDetailGroupStore{}
	controller := &mockExpenseController{}

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
				GroupID:        mockGroupID,
				CreateByUserID: mockCreatorID,
				ExpenseTypeID:  mockExpenseTypeID,
			},
			expenseID:        mockExpenseID.String(),
			expectFail:       false,
			expectStatusCode: http.StatusCreated,
		},
		{
			name: "invalid expense id",
			payload: types.ExpenseUpdatePayload{
				GroupID:        mockGroupID,
				CreateByUserID: mockCreatorID,
				ExpenseTypeID:  mockExpenseTypeID,
			},
			expenseID:        uuid.NewString(),
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalid group id",
			payload: types.ExpenseUpdatePayload{
				GroupID:        uuid.New(),
				CreateByUserID: mockCreatorID,
				ExpenseTypeID:  mockExpenseTypeID,
			},
			expenseID:        mockExpenseID.String(),
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
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
			router.PUT("/expense/:expenseId", handler.handleUpdateExpense)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}

type mockUpdateExpenseDetailStore struct{}

func (s *mockUpdateExpenseDetailStore) CreateExpense(expense types.Expense) error {
	return nil
}
func (s *mockUpdateExpenseDetailStore) CreateItem(item types.Item) error {
	return nil
}
func (s *mockUpdateExpenseDetailStore) CreateLedger(ledger types.Ledger) error {
	return nil
}
func (s *mockUpdateExpenseDetailStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	if expenseID != mockExpenseID.String() {
		return nil, types.ErrExpenseNotExist
	}
	expense := &types.Expense{
		ID:      mockExpenseID,
		GroupID: mockGroupID,
	}
	return expense, nil
}
func (s *mockUpdateExpenseDetailStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	return nil, nil
}
func (s *mockUpdateExpenseDetailStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return mockItems, nil
}
func (s *mockUpdateExpenseDetailStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockUpdateExpenseDetailStore) GetLedgerUnsettledFromGroup(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockUpdateExpenseDetailStore) UpdateExpense(expense types.Expense) error {
	return nil
}
func (s *mockUpdateExpenseDetailStore) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}
func (s *mockUpdateExpenseDetailStore) UpdateItem(item types.Item) error {
	return nil
}
func (s *mockUpdateExpenseDetailStore) UpdateLedger(ledger types.Ledger) error {
	return nil
}

type mockUpdateExpenseDetailGroupStore struct{}

func (m *mockUpdateExpenseDetailGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockUpdateExpenseDetailGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockUpdateExpenseDetailGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if groupID != mockGroupID.String() {
		return nil, types.ErrGroupNotExist
	}
	if userID != mockUserID.String() {
		return nil, types.ErrUserNotPermitted
	}
	return nil, nil
}
func (m *mockUpdateExpenseDetailGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockUpdateExpenseDetailGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockUpdateExpenseDetailGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockUpdateExpenseDetailGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockUpdateExpenseDetailGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}

type mockUpdateExpenseDetailUserStore struct{}

func (m *mockUpdateExpenseDetailUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockUpdateExpenseDetailUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockUpdateExpenseDetailUserStore) GetUserByID(id string) (*types.User, error) {
	user := &types.User{
		ID:       mockUserID,
		Username: "test user",
	}
	return user, nil
}
func (m *mockUpdateExpenseDetailUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockUpdateExpenseDetailUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
