package expense

import (
	"expense-tracker/types"

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
