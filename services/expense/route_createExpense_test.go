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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestRouteCreateExpense(t *testing.T) {
	store := &mockCreateExpenseStore{}
	userStore := &mockCreateExpenseUserStore{}
	groupStore := &mockCreateExpenseGroupStore{}
	controller := &mockExpenseController{}

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		payload          types.ExpensePayload
		expectFail       bool
		expectStatusCode int
	}

	subtests := []testcase{
		{
			name: "valid",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: mockCreatorID.String(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       false,
			expectStatusCode: http.StatusCreated,
		},
		{
			name: "invalid mismatch user id and creator id",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: uuid.NewString(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			marshalled, _ := json.Marshal(test.payload)
			req, err := http.NewRequest(http.MethodPost, "/create_expense", bytes.NewBuffer(marshalled))
			if err != nil {
				t.Fatal(err)
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
			router.POST("/create_expense", handler.handleCreateExpense)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}

var mockUserID = uuid.New()
var mockGroupID = uuid.New()
var mockCreatorID = mockUserID
var mockPayerID = uuid.New()
var mockExpenseTypeID = uuid.New()

type mockCreateExpenseStore struct{}

func (s *mockCreateExpenseStore) CreateExpense(expense types.Expense) error {
	return nil
}
func (s *mockCreateExpenseStore) CreateItem(item types.Item) error {
	return nil
}
func (s *mockCreateExpenseStore) CreateLedger(ledger types.Ledger) error {
	return nil
}
func (s *mockCreateExpenseStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	return nil, nil
}
func (s *mockCreateExpenseStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	return nil, nil
}
func (s *mockCreateExpenseStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return nil, nil
}
func (s *mockCreateExpenseStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockCreateExpenseStore) GetLedgerUnsettledFromGroup(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockCreateExpenseStore) GetExpenseType() ([]*types.ExpenseType, error) {
	return nil, nil
}

func (s *mockCreateExpenseStore) UpdateExpense(expense types.Expense) error {
	return nil
}
func (s *mockCreateExpenseStore) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}
func (s *mockCreateExpenseStore) UpdateItem(item types.Item) error {
	return nil
}
func (s *mockCreateExpenseStore) UpdateLedger(ledger types.Ledger) error {
	return nil
}

type mockCreateExpenseGroupStore struct{}

func (m *mockCreateExpenseGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockCreateExpenseGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockCreateExpenseGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if groupID != mockGroupID.String() {
		return nil, types.ErrGroupNotExist
	}
	if userID != mockUserID.String() {
		return nil, types.ErrUserNotPermitted
	}
	return nil, nil
}
func (m *mockCreateExpenseGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockCreateExpenseGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockCreateExpenseGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockCreateExpenseGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockCreateExpenseGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockCreateExpenseGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}

type mockCreateExpenseUserStore struct{}

func (m *mockCreateExpenseUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockCreateExpenseUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockCreateExpenseUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockCreateExpenseUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockCreateExpenseUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockCreateExpenseUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockCreateExpenseUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockCreateExpenseUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockCreateExpenseUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockExpenseController struct{}

func (m *mockExpenseController) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	return nil
}
