package expense

import (
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

func TestRouteSettleExpense(t *testing.T) {
	store := &mockSettelExpenseStore{}
	userStore := &mockSettelExpenseUserStore{}
	groupStore := &mockUSettelExpenseGroupStore{}
	controller := &mockExpenseController{}

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
			router.PUT("/settle_expense/:groupId", handler.handleSettleExpense)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}

type mockSettelExpenseStore struct{}

func (s *mockSettelExpenseStore) CreateExpense(expense types.Expense) error {
	return nil
}
func (s *mockSettelExpenseStore) CreateItem(item types.Item) error {
	return nil
}
func (s *mockSettelExpenseStore) CreateLedger(ledger types.Ledger) error {
	return nil
}
func (s *mockSettelExpenseStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	return nil, nil
}
func (s *mockSettelExpenseStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	return nil, nil
}
func (s *mockSettelExpenseStore) GetExpenseType() ([]*types.ExpenseType, error) {
	return nil, nil
}
func (s *mockSettelExpenseStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return nil, nil
}
func (s *mockSettelExpenseStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockSettelExpenseStore) GetLedgerUnsettledFromGroup(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockSettelExpenseStore) UpdateExpense(expense types.Expense) error {
	return nil
}
func (s *mockSettelExpenseStore) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}
func (s *mockSettelExpenseStore) UpdateItem(item types.Item) error {
	return nil
}
func (s *mockSettelExpenseStore) UpdateLedger(ledger types.Ledger) error {
	return nil
}

type mockUSettelExpenseGroupStore struct{}

func (m *mockUSettelExpenseGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockUSettelExpenseGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockUSettelExpenseGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if groupID != mockGroupID.String() {
		return nil, types.ErrGroupNotExist
	}
	if userID != mockUserID.String() {
		return nil, types.ErrUserNotPermitted
	}
	return nil, nil
}
func (m *mockUSettelExpenseGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockUSettelExpenseGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockUSettelExpenseGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockUSettelExpenseGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockUSettelExpenseGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockUSettelExpenseGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}

type mockSettelExpenseUserStore struct{}

func (m *mockSettelExpenseUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockSettelExpenseUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockSettelExpenseUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockSettelExpenseUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockSettelExpenseUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockSettelExpenseUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockSettelExpenseUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockSettelExpenseUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockSettelExpenseUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}
