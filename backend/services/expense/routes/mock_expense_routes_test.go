package expense

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

var mockUserID = uuid.New()
var mockGroupID = uuid.New()
var mockCreatorID = uuid.New()
var mockPayerID = uuid.New()
var mockExpenseTypeID = uuid.New()

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

// expense store interface
type mockExpenseStore struct{}

func (s *mockExpenseStore) CreateExpense(expense types.Expense) error {
	return nil
}
func (s *mockExpenseStore) CreateItem(item types.Item) error {
	return nil
}
func (s *mockExpenseStore) CreateLedger(ledger types.Ledger) error {
	return nil
}
func (s *mockExpenseStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	return nil, nil
}
func (s *mockExpenseStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	return nil, nil
}
func (s *mockExpenseStore) GetExpenseType() ([]*types.ExpenseType, error) {
	return nil, nil
}
func (s *mockExpenseStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return nil, nil
}
func (s *mockExpenseStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockExpenseStore) GetLedgerUnsettledFromGroup(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}
func (s *mockExpenseStore) UpdateExpense(expense types.Expense) error {
	return nil
}
func (s *mockExpenseStore) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}
func (s *mockExpenseStore) UpdateItem(item types.Item) error {
	return nil
}
func (s *mockExpenseStore) UpdateLedger(ledger types.Ledger) error {
	return nil
}
func (m *mockExpenseStore) CheckExpenseExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockExpenseStore) GetExpenseTypeById(id uuid.UUID) (string, error) {
	return "", nil
}
func (m *mockExpenseStore) DeleteExpense(expense types.Expense) error {
	return nil
}
func (m *mockExpenseStore) CreateBalances(groupId string, balances []*types.Balance) error {
	return nil
}
func (m *mockExpenseStore) CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error {
	return nil
}
func (m *mockExpenseStore) OutdateBalanceByGroupId(groupId string) error {
	return nil
}
func (m *mockExpenseStore) GetBalanceByGroupId(groupId string) ([]types.Balance, error) {
	return nil, nil
}

func (m *mockExpenseStore) SettleExpenseByGroupId(groupId string) error {
	return nil
}
func (m *mockExpenseStore) CheckBalanceExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockExpenseStore) SettleBalanceByBalanceId(balanceId string) error {
	return nil
}
func (m *mockExpenseStore) CheckGroupBallanceAllSettled(groupId string) (bool, error) {
	return false, nil
}

// group store interface
type mockGroupStore struct{}

func (m *mockGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockGroupStore) CheckGroupExistById(id string) (bool, error) {
	return false, nil
}
func (m *mockGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	return false, nil
}

// user store interface
type mockUserStore struct{}

func (m *mockUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

// controller interface
type mockController struct{}

func (m *mockController) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	return nil
}

type mockCreateExpenseStore struct {
	mockExpenseStore
}

type mockCreateExpenseGroupStore struct {
	mockGroupStore
}

func (m *mockCreateExpenseGroupStore) CheckGroupExistById(id string) (bool, error) {
	if id == mockGroupID.String() {
		return true, nil
	}
	return false, nil
}
func (m *mockCreateExpenseGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if (groupId == mockGroupID.String()) && (userId == mockCreatorID.String()) {
		return true, nil
	}
	return false, nil
}

type mockCreateExpenseUserStore struct {
	mockUserStore
}

func (m *mockCreateExpenseUserStore) CheckUserExistByID(id string) (bool, error) {
	if id == mockCreatorID.String() || id == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockExpenseController struct {
	mockController
}

type mockUpdateExpenseDetailStore struct {
	mockExpenseStore
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
func (m *mockUpdateExpenseDetailStore) CheckExpenseExistByID(id string) (bool, error) {
	if id == mockExpenseID.String() {
		return true, nil
	}
	return false, nil
}

type mockUpdateExpenseDetailGroupStore struct {
	mockGroupStore
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
func (m *mockUpdateExpenseDetailGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockUpdateExpenseDetailUserStore struct {
	mockUserStore
}

func (m *mockUpdateExpenseDetailUserStore) GetUserByID(id string) (*types.User, error) {
	user := &types.User{
		ID:       mockUserID,
		Username: "test user",
	}
	return user, nil
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

type mockSettelExpenseStore struct {
	mockExpenseStore
}

type mockUSettelExpenseGroupStore struct {
	mockGroupStore
}

func (m *mockUSettelExpenseGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockSettelExpenseUserStore struct {
	mockUserStore
}

type mockGetUnsettledBalanceStore struct {
	mockExpenseStore
}

func (s *mockGetUnsettledBalanceStore) GetBalanceByGroupId(groupId string) ([]types.Balance, error) {
	if groupId != mockGroupID.String() {
		return nil, nil
	}
	balances := make([]types.Balance, 0, len(mockBalance))
	for _, balance := range mockBalance {
		balances = append(balances, *balance)
	}
	return balances, nil
}

func (s *mockGetUnsettledBalanceStore) GetLedgerUnsettledFromGroup(groupID string) ([]*types.Ledger, error) {
	if groupID != mockGroupID.String() {
		return nil, nil
	}

	return mockLedger, nil
}

type mockGetUnsettledBalanceGroupStore struct {
	mockGroupStore
}

func (m *mockGetUnsettledBalanceGroupStore) GetGroupCurrency(groupID string) (string, error) {
	if groupID != mockGroupID.String() {
		return "", nil
	}
	return mockCurrency, nil
}

func (m *mockGetUnsettledBalanceGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockGetUnsettledBalanceUserStore struct {
	mockUserStore
}

type mockGetUnsettledBalanceController struct {
	mockExpenseController
}

func (c *mockGetUnsettledBalanceController) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	return mockBalance
}

type mockGetExpenseDetailStore struct {
	mockExpenseStore
}

func (s *mockGetExpenseDetailStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	expense := &types.Expense{
		ID:      mockExpenseID,
		GroupID: mockGroupID,
	}
	return expense, nil
}

func (s *mockGetExpenseDetailStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return mockItems, nil
}
func (m *mockGetExpenseDetailStore) CheckExpenseExistByID(id string) (bool, error) {
	if id == mockExpenseID.String() {
		return true, nil
	}
	return false, nil
}

type mockGetExpenseDetailGroupStore struct {
	mockGroupStore
}

func (m *mockGetExpenseDetailGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockGetExpenseDetailUserStore struct {
	mockUserStore
}

func (m *mockGetExpenseDetailUserStore) GetUserByID(id string) (*types.User, error) {
	user := &types.User{
		ID:       mockUserID,
		Username: "test user",
	}
	return user, nil
}
