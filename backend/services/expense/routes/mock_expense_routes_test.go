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

// expense store base mock

type mockExpenseStore struct {
	CreateExpenseFn             func(expense types.Expense) error
	CreateItemFn                func(item types.Item) error
	CreateLedgerFn              func(ledger types.Ledger) error
	GetExpenseByIDFn            func(expenseID string) (*types.Expense, error)
	GetExpenseListFn            func(groupID string, page int64) ([]*types.Expense, error)
	GetExpenseTypeFn            func() ([]*types.ExpenseType, error)
	GetItemsByExpenseIDFn       func(expenseID string) ([]*types.Item, error)
	GetLedgersByExpenseIDFn     func(expenseID string) ([]*types.Ledger, error)
	GetLedgerUnsettledFromGroupFn func(expenseID string) ([]*types.Ledger, error)
	UpdateExpenseFn             func(expense types.Expense) error
	UpdateExpenseSettleInGroupFn func(groupID string) error
	UpdateItemFn                func(item types.Item) error
	UpdateLedgerFn              func(ledger types.Ledger) error
	CheckExpenseExistByIDFn     func(id string) (bool, error)
	GetExpenseTypeByIdFn        func(id uuid.UUID) (string, error)
	DeleteExpenseFn             func(expense types.Expense) error
	CreateBalancesFn            func(groupId string, balances []*types.Balance) error
	CreateBalanceLedgerFn       func(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error
	OutdateBalanceByGroupIdFn   func(groupId string) error
	GetBalanceByGroupIdFn       func(groupId string) ([]types.Balance, error)
	SettleExpenseByGroupIdFn    func(groupId string) error
	CheckBalanceExistByIDFn     func(id string) (bool, error)
	SettleBalanceByBalanceIdFn  func(balanceId string) error
	CheckGroupBallanceAllSettledFn func(groupId string) (bool, error)
}

func (s *mockExpenseStore) CreateExpense(expense types.Expense) error {
	if s.CreateExpenseFn != nil {
		return s.CreateExpenseFn(expense)
	}
	return nil
}
func (s *mockExpenseStore) CreateItem(item types.Item) error {
	if s.CreateItemFn != nil {
		return s.CreateItemFn(item)
	}
	return nil
}
func (s *mockExpenseStore) CreateLedger(ledger types.Ledger) error {
	if s.CreateLedgerFn != nil {
		return s.CreateLedgerFn(ledger)
	}
	return nil
}
func (s *mockExpenseStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	if s.GetExpenseByIDFn != nil {
		return s.GetExpenseByIDFn(expenseID)
	}
	return nil, nil
}
func (s *mockExpenseStore) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	if s.GetExpenseListFn != nil {
		return s.GetExpenseListFn(groupID, page)
	}
	return nil, nil
}
func (s *mockExpenseStore) GetExpenseType() ([]*types.ExpenseType, error) {
	if s.GetExpenseTypeFn != nil {
		return s.GetExpenseTypeFn()
	}
	return nil, nil
}
func (s *mockExpenseStore) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	if s.GetItemsByExpenseIDFn != nil {
		return s.GetItemsByExpenseIDFn(expenseID)
	}
	return nil, nil
}
func (s *mockExpenseStore) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	if s.GetLedgersByExpenseIDFn != nil {
		return s.GetLedgersByExpenseIDFn(expenseID)
	}
	return nil, nil
}
func (s *mockExpenseStore) GetLedgerUnsettledFromGroup(expenseID string) ([]*types.Ledger, error) {
	if s.GetLedgerUnsettledFromGroupFn != nil {
		return s.GetLedgerUnsettledFromGroupFn(expenseID)
	}
	return nil, nil
}
func (s *mockExpenseStore) UpdateExpense(expense types.Expense) error {
	if s.UpdateExpenseFn != nil {
		return s.UpdateExpenseFn(expense)
	}
	return nil
}
func (s *mockExpenseStore) UpdateExpenseSettleInGroup(groupID string) error {
	if s.UpdateExpenseSettleInGroupFn != nil {
		return s.UpdateExpenseSettleInGroupFn(groupID)
	}
	return nil
}
func (s *mockExpenseStore) UpdateItem(item types.Item) error {
	if s.UpdateItemFn != nil {
		return s.UpdateItemFn(item)
	}
	return nil
}
func (s *mockExpenseStore) UpdateLedger(ledger types.Ledger) error {
	if s.UpdateLedgerFn != nil {
		return s.UpdateLedgerFn(ledger)
	}
	return nil
}
func (s *mockExpenseStore) CheckExpenseExistByID(id string) (bool, error) {
	if s.CheckExpenseExistByIDFn != nil {
		return s.CheckExpenseExistByIDFn(id)
	}
	return false, nil
}
func (s *mockExpenseStore) GetExpenseTypeById(id uuid.UUID) (string, error) {
	if s.GetExpenseTypeByIdFn != nil {
		return s.GetExpenseTypeByIdFn(id)
	}
	return "", nil
}
func (s *mockExpenseStore) DeleteExpense(expense types.Expense) error {
	if s.DeleteExpenseFn != nil {
		return s.DeleteExpenseFn(expense)
	}
	return nil
}
func (s *mockExpenseStore) CreateBalances(groupId string, balances []*types.Balance) error {
	if s.CreateBalancesFn != nil {
		return s.CreateBalancesFn(groupId, balances)
	}
	return nil
}
func (s *mockExpenseStore) CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error {
	if s.CreateBalanceLedgerFn != nil {
		return s.CreateBalanceLedgerFn(balanceIds, ledgerIds)
	}
	return nil
}
func (s *mockExpenseStore) OutdateBalanceByGroupId(groupId string) error {
	if s.OutdateBalanceByGroupIdFn != nil {
		return s.OutdateBalanceByGroupIdFn(groupId)
	}
	return nil
}
func (s *mockExpenseStore) GetBalanceByGroupId(groupId string) ([]types.Balance, error) {
	if s.GetBalanceByGroupIdFn != nil {
		return s.GetBalanceByGroupIdFn(groupId)
	}
	return nil, nil
}
func (s *mockExpenseStore) SettleExpenseByGroupId(groupId string) error {
	if s.SettleExpenseByGroupIdFn != nil {
		return s.SettleExpenseByGroupIdFn(groupId)
	}
	return nil
}
func (s *mockExpenseStore) CheckBalanceExistByID(id string) (bool, error) {
	if s.CheckBalanceExistByIDFn != nil {
		return s.CheckBalanceExistByIDFn(id)
	}
	return false, nil
}
func (s *mockExpenseStore) SettleBalanceByBalanceId(balanceId string) error {
	if s.SettleBalanceByBalanceIdFn != nil {
		return s.SettleBalanceByBalanceIdFn(balanceId)
	}
	return nil
}
func (s *mockExpenseStore) CheckGroupBallanceAllSettled(groupId string) (bool, error) {
	if s.CheckGroupBallanceAllSettledFn != nil {
		return s.CheckGroupBallanceAllSettledFn(groupId)
	}
	return false, nil
}

// group store base mock

type mockGroupStore struct {
	CreateGroupFn          func(group types.Group) error
	GetGroupByIDFn         func(id string) (*types.Group, error)
	GetGroupByIDAndUserFn  func(groupID string, userID string) (*types.Group, error)
	GetGroupListByUserFn   func(userid string) ([]*types.Group, error)
	GetGroupMemberByGroupIDFn func(groupId string) ([]*types.User, error)
	UpdateGroupMemberFn    func(action string, userid string, groupid string) error
	UpdateGroupStatusFn    func(groupid string, isActive bool) error
	GetGroupCurrencyFn     func(groupID string) (string, error)
	GetRelatedUserFn       func(currentUser string, groupId string) ([]*types.RelatedMember, error)
	CheckGroupExistByIdFn  func(id string) (bool, error)
	CheckGroupUserPairExistFn func(groupId string, userId string) (bool, error)
}

func (m *mockGroupStore) CreateGroup(group types.Group) error {
	if m.CreateGroupFn != nil {
		return m.CreateGroupFn(group)
	}
	return nil
}
func (m *mockGroupStore) GetGroupByID(id string) (*types.Group, error) {
	if m.GetGroupByIDFn != nil {
		return m.GetGroupByIDFn(id)
	}
	return nil, nil
}
func (m *mockGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if m.GetGroupByIDAndUserFn != nil {
		return m.GetGroupByIDAndUserFn(groupID, userID)
	}
	return nil, nil
}
func (m *mockGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	if m.GetGroupListByUserFn != nil {
		return m.GetGroupListByUserFn(userid)
	}
	return nil, nil
}
func (m *mockGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	if m.GetGroupMemberByGroupIDFn != nil {
		return m.GetGroupMemberByGroupIDFn(groupId)
	}
	return nil, nil
}
func (m *mockGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	if m.UpdateGroupMemberFn != nil {
		return m.UpdateGroupMemberFn(action, userid, groupid)
	}
	return nil
}
func (m *mockGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	if m.UpdateGroupStatusFn != nil {
		return m.UpdateGroupStatusFn(groupid, isActive)
	}
	return nil
}
func (m *mockGroupStore) GetGroupCurrency(groupID string) (string, error) {
	if m.GetGroupCurrencyFn != nil {
		return m.GetGroupCurrencyFn(groupID)
	}
	return "", nil
}
func (m *mockGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	if m.GetRelatedUserFn != nil {
		return m.GetRelatedUserFn(currentUser, groupId)
	}
	return nil, nil
}
func (m *mockGroupStore) CheckGroupExistById(id string) (bool, error) {
	if m.CheckGroupExistByIdFn != nil {
		return m.CheckGroupExistByIdFn(id)
	}
	return false, nil
}
func (m *mockGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if m.CheckGroupUserPairExistFn != nil {
		return m.CheckGroupUserPairExistFn(groupId, userId)
	}
	return false, nil
}

// user store base mock

type mockUserStore struct {
	GetUserByEmailFn        func(email string) (*types.User, error)
	GetUserByUsernameFn     func(username string) (*types.User, error)
	GetUserByIDFn           func(id string) (*types.User, error)
	CreateUserFn            func(user types.User) error
	GetUsernameByIDFn       func(userid string) (string, error)
	CheckEmailExistFn       func(email string) (bool, error)
	CheckUserExistByEmailFn func(email string) (bool, error)
	CheckUserExistByIDFn    func(id string) (bool, error)
	CheckUserExistByUserFn  func(username string) (bool, error)
}

func (m *mockUserStore) GetUserByEmail(email string) (*types.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(email)
	}
	return nil, nil
}
func (m *mockUserStore) GetUserByUsername(username string) (*types.User, error) {
	if m.GetUserByUsernameFn != nil {
		return m.GetUserByUsernameFn(username)
	}
	return nil, nil
}
func (m *mockUserStore) GetUserByID(id string) (*types.User, error) {
	if m.GetUserByIDFn != nil {
		return m.GetUserByIDFn(id)
	}
	return nil, nil
}
func (m *mockUserStore) CreateUser(user types.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(user)
	}
	return nil
}
func (m *mockUserStore) GetUsernameByID(userid string) (string, error) {
	if m.GetUsernameByIDFn != nil {
		return m.GetUsernameByIDFn(userid)
	}
	return "", nil
}
func (m *mockUserStore) CheckEmailExist(email string) (bool, error) {
	if m.CheckEmailExistFn != nil {
		return m.CheckEmailExistFn(email)
	}
	return false, nil
}
func (m *mockUserStore) CheckUserExistByEmail(email string) (bool, error) {
	if m.CheckUserExistByEmailFn != nil {
		return m.CheckUserExistByEmailFn(email)
	}
	return false, nil
}
func (m *mockUserStore) CheckUserExistByID(id string) (bool, error) {
	if m.CheckUserExistByIDFn != nil {
		return m.CheckUserExistByIDFn(id)
	}
	return false, nil
}
func (m *mockUserStore) CheckUserExistByUsername(username string) (bool, error) {
	if m.CheckUserExistByUserFn != nil {
		return m.CheckUserExistByUserFn(username)
	}
	return false, nil
}

// controller base mock

type mockController struct {
	DebtSimplifyFn func(ledgers []*types.Ledger) []*types.Balance
}

func (m *mockController) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	if m.DebtSimplifyFn != nil {
		return m.DebtSimplifyFn(ledgers)
	}
	return nil
}

func expenseStoreMock() *mockExpenseStore { return &mockExpenseStore{} }
func groupStoreMock() *mockGroupStore     { return &mockGroupStore{} }
func userStoreMock() *mockUserStore       { return &mockUserStore{} }
func expenseControllerMock() *mockController {
	return &mockController{}
}

func createExpenseStoreMock() *mockExpenseStore { return expenseStoreMock() }

func createExpenseGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.CheckGroupExistByIdFn = func(id string) (bool, error) {
		return id == mockGroupID.String(), nil
	}
	store.CheckGroupUserPairExistFn = func(groupId string, userId string) (bool, error) {
		return groupId == mockGroupID.String() && userId == mockCreatorID.String(), nil
	}
	return store
}

func createExpenseUserStoreMock() *mockUserStore {
	store := userStoreMock()
	store.CheckUserExistByIDFn = func(id string) (bool, error) {
		return id == mockCreatorID.String() || id == mockUserID.String(), nil
	}
	return store
}

func updateExpenseDetailStoreMock() *mockExpenseStore {
	store := expenseStoreMock()
	store.GetExpenseByIDFn = func(expenseID string) (*types.Expense, error) {
		if expenseID != mockExpenseID.String() {
			return nil, types.ErrExpenseNotExist
		}
		return &types.Expense{ID: mockExpenseID, GroupID: mockGroupID}, nil
	}
	store.CheckExpenseExistByIDFn = func(id string) (bool, error) {
		return id == mockExpenseID.String(), nil
	}
	return store
}

func updateExpenseDetailGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.GetGroupByIDAndUserFn = func(groupID string, userID string) (*types.Group, error) {
		if groupID != mockGroupID.String() {
			return nil, types.ErrGroupNotExist
		}
		if userID != mockUserID.String() {
			return nil, types.ErrUserNotPermitted
		}
		return &types.Group{}, nil
	}
	store.CheckGroupUserPairExistFn = func(groupId string, userId string) (bool, error) {
		return groupId == mockGroupID.String() && userId == mockUserID.String(), nil
	}
	return store
}

func updateExpenseDetailUserStoreMock() *mockUserStore {
	store := userStoreMock()
	store.GetUserByIDFn = func(id string) (*types.User, error) {
		return &types.User{ID: mockUserID, Username: "test user"}, nil
	}
	return store
}

func getExpenseListStoreMock() *mockExpenseStore {
	store := expenseStoreMock()
	store.GetExpenseListFn = func(groupID string, page int64) ([]*types.Expense, error) {
		if page > int64(mockTotalPage) {
			return nil, types.ErrNoRemainingExpenses
		}
		return []*types.Expense{{ID: mockExpenseIDs[0]}, {ID: mockExpenseIDs[1]}, {ID: mockExpenseIDs[2]}}, nil
	}
	return store
}

func getExpenseListGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.CheckGroupExistByIdFn = func(id string) (bool, error) {
		return id == mockGroupID.String(), nil
	}
	store.CheckGroupUserPairExistFn = func(groupId string, userId string) (bool, error) {
		return groupId == mockGroupID.String() && userId == mockUserID.String(), nil
	}
	return store
}

func getExpenseListUserStoreMock() *mockUserStore { return userStoreMock() }

func settleExpenseStoreMock() *mockExpenseStore { return expenseStoreMock() }

func settleExpenseGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.CheckGroupUserPairExistFn = func(groupId string, userId string) (bool, error) {
		return groupId == mockGroupID.String() && userId == mockUserID.String(), nil
	}
	return store
}

func settleExpenseUserStoreMock() *mockUserStore { return userStoreMock() }

func getUnsettledBalanceStoreMock() *mockExpenseStore {
	store := expenseStoreMock()
	store.GetBalanceByGroupIdFn = func(groupId string) ([]types.Balance, error) {
		if groupId != mockGroupID.String() {
			return nil, nil
		}
		balances := make([]types.Balance, 0, len(mockBalance))
		for _, balance := range mockBalance {
			balances = append(balances, *balance)
		}
		return balances, nil
	}
	store.GetLedgerUnsettledFromGroupFn = func(groupID string) ([]*types.Ledger, error) {
		if groupID != mockGroupID.String() {
			return nil, nil
		}
		return mockLedger, nil
	}
	return store
}

func getUnsettledBalanceGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.GetGroupCurrencyFn = func(groupID string) (string, error) {
		if groupID != mockGroupID.String() {
			return "", nil
		}
		return mockCurrency, nil
	}
	store.CheckGroupUserPairExistFn = func(groupId string, userId string) (bool, error) {
		return groupId == mockGroupID.String() && userId == mockUserID.String(), nil
	}
	return store
}

func getUnsettledBalanceUserStoreMock() *mockUserStore { return userStoreMock() }

func getUnsettledBalanceControllerMock() *mockController {
	controller := expenseControllerMock()
	controller.DebtSimplifyFn = func(ledgers []*types.Ledger) []*types.Balance {
		return mockBalance
	}
	return controller
}

func getExpenseDetailStoreMock() *mockExpenseStore {
	store := expenseStoreMock()
	store.GetExpenseByIDFn = func(expenseID string) (*types.Expense, error) {
		return &types.Expense{ID: mockExpenseID, GroupID: mockGroupID}, nil
	}
	store.GetItemsByExpenseIDFn = func(expenseID string) ([]*types.Item, error) {
		return mockItems, nil
	}
	store.CheckExpenseExistByIDFn = func(id string) (bool, error) {
		return id == mockExpenseID.String(), nil
	}
	return store
}

func getExpenseDetailGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.CheckGroupUserPairExistFn = func(groupId string, userId string) (bool, error) {
		return groupId == mockGroupID.String() && userId == mockUserID.String(), nil
	}
	return store
}

func getExpenseDetailUserStoreMock() *mockUserStore {
	store := userStoreMock()
	store.GetUserByIDFn = func(id string) (*types.User, error) {
		return &types.User{ID: mockUserID, Username: "test user"}, nil
	}
	return store
}
