package tracker

import (
	"database/sql"
	"errors"
	"net/http"
	"testing"
	"time"

	"expense-tracker/backend/config"
	expense "expense-tracker/backend/services/expense"
	expensestore "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type groupSettlementFixture struct {
	actorID                 uuid.UUID
	counterpartyID          uuid.UUID
	nonPartyID              uuid.UUID
	groupID                 uuid.UUID
	unrelatedGroupID        uuid.UUID
	openExpenseID           uuid.UUID
	deletedExpenseID        uuid.UUID
	unrelatedOpenExpenseID  uuid.UUID
	openLedgerID            uuid.UUID
	deletedLedgerID         uuid.UUID
	unrelatedOpenLedgerID   uuid.UUID
	currentBalanceID        uuid.UUID
	historicalBalanceID     uuid.UUID
	unrelatedCurrentBalance uuid.UUID
}

func TestGroupSettlementPersistenceAndRetry(t *testing.T) {
	database := openAuthorizationTestDB(t)
	fixture := createGroupSettlementFixture(t, database)
	handler := NewHandler(database)

	response := serveAuthenticatedRequest(t, handler, http.MethodPut,
		config.Envs.APIPath+"/settle_expense/"+fixture.groupID.String(), fixture.actorID)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	assertExpenseSettlementState(t, database, fixture.openExpenseID, true)
	assertExpenseSettlementState(t, database, fixture.deletedExpenseID, false)
	assertExpenseSettlementState(t, database, fixture.unrelatedOpenExpenseID, false)
	assertBalanceOutdatedState(t, database, fixture.currentBalanceID, true)
	assertBalanceOutdatedState(t, database, fixture.unrelatedCurrentBalance, false)
	openSettledAt := readExpenseSettlementTime(t, database, fixture.openExpenseID)
	currentBalanceUpdatedAt := readBalanceUpdateTime(t, database, fixture.currentBalanceID)

	retryResponse := serveAuthenticatedRequest(t, handler, http.MethodPut,
		config.Envs.APIPath+"/settle_expense/"+fixture.groupID.String(), fixture.actorID)
	require.Equal(t, http.StatusCreated, retryResponse.Code, retryResponse.Body.String())
	require.Equal(t, openSettledAt, readExpenseSettlementTime(t, database, fixture.openExpenseID))
	require.Equal(t, currentBalanceUpdatedAt, readBalanceUpdateTime(t, database, fixture.currentBalanceID))
	assertExpenseSettlementState(t, database, fixture.deletedExpenseID, false)
}

func TestIndividualBalanceSettlementRequiresParty(t *testing.T) {
	database := openAuthorizationTestDB(t)
	fixture := createGroupSettlementFixture(t, database)
	handler := NewHandler(database)

	response := serveAuthenticatedRequest(t, handler, http.MethodPost,
		config.Envs.APIPath+"/settle_balance/"+fixture.groupID.String()+"/"+fixture.currentBalanceID.String(), fixture.nonPartyID)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())

	assertBalanceSettlementState(t, database, fixture.currentBalanceID, false)
	assertExpenseSettlementState(t, database, fixture.openExpenseID, false)
}

func TestFinalCurrentBalanceIgnoresOutdatedHistory(t *testing.T) {
	database := openAuthorizationTestDB(t)
	fixture := createGroupSettlementFixture(t, database)
	handler := NewHandler(database)

	response := serveAuthenticatedRequest(t, handler, http.MethodPost,
		config.Envs.APIPath+"/settle_balance/"+fixture.groupID.String()+"/"+fixture.currentBalanceID.String(), fixture.actorID)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	assertBalanceSettlementState(t, database, fixture.currentBalanceID, true)
	assertBalanceSettlementState(t, database, fixture.historicalBalanceID, false)
	assertExpenseSettlementState(t, database, fixture.openExpenseID, true)
	assertExpenseSettlementState(t, database, fixture.deletedExpenseID, false)
}

func TestArchivedGroupCannotBeSettled(t *testing.T) {
	database := openAuthorizationTestDB(t)
	fixture := createGroupSettlementFixture(t, database)
	handler := NewHandler(database)
	_, err := database.Exec("UPDATE groups SET is_active = FALSE WHERE id = $1", fixture.groupID)
	require.NoError(t, err)

	response := serveAuthenticatedRequest(t, handler, http.MethodPut,
		config.Envs.APIPath+"/settle_expense/"+fixture.groupID.String(), fixture.actorID)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())

	assertExpenseSettlementState(t, database, fixture.openExpenseID, false)
	assertBalanceOutdatedState(t, database, fixture.currentBalanceID, false)
}

func TestSettlementRecoveryReconciliation(t *testing.T) {
	database := openAuthorizationTestDB(t)
	fixture := createGroupSettlementFixture(t, database)
	store := expensestore.NewStore(database)
	injectedErr := errors.New("simulated response-stage failure")

	initial := reconcileSettlementFixture(t, store, fixture.groupID)
	require.Equal(t, expense.SettlementNotStarted, initial.State)

	err := store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		if _, err := transactionStore.LockGroupCurrency(fixture.groupID.String()); err != nil {
			return err
		}
		if err := transactionStore.UpdateExpenseSettleInGroup(fixture.groupID.String()); err != nil {
			return err
		}
		if err := transactionStore.OutdateBalanceByGroupId(fixture.groupID.String()); err != nil {
			return err
		}
		return injectedErr
	})
	require.ErrorIs(t, err, injectedErr)
	require.Equal(t, expense.SettlementNotStarted, reconcileSettlementFixture(t, store, fixture.groupID).State)

	// Treat the successful response as lost and determine the outcome only from
	// authoritative persisted state.
	handler := NewHandler(database)
	_ = serveAuthenticatedRequest(t, handler, http.MethodPut,
		config.Envs.APIPath+"/settle_expense/"+fixture.groupID.String(), fixture.actorID)
	committed := reconcileSettlementFixture(t, store, fixture.groupID)
	require.Equal(t, expense.SettlementCommitted, committed.State)
	require.Zero(t, committed.UnsettledLedgerCount)
	require.Zero(t, committed.CurrentBalanceCount)
}

func TestSettlementReconciliationDetectsMissingCurrentBalance(t *testing.T) {
	database := openAuthorizationTestDB(t)
	fixture := createGroupSettlementFixture(t, database)
	store := expensestore.NewStore(database)

	require.NoError(t, store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		if _, err := transactionStore.LockGroupCurrency(fixture.groupID.String()); err != nil {
			return err
		}
		return transactionStore.OutdateBalanceByGroupId(fixture.groupID.String())
	}))

	result := reconcileSettlementFixture(t, store, fixture.groupID)
	require.Equal(t, expense.SettlementInconsistent, result.State)
	require.Positive(t, result.UnsettledLedgerCount)
	require.Zero(t, result.CurrentBalanceCount)
}

func reconcileSettlementFixture(t *testing.T, store *expensestore.Store, groupID uuid.UUID) expense.SettlementReconciliation {
	t.Helper()

	var result expense.SettlementReconciliation
	require.NoError(t, store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		if _, err := transactionStore.LockGroupCurrency(groupID.String()); err != nil {
			return err
		}
		var err error
		result, err = expense.ReconcileGroupSettlement(transactionStore, expense.NewController(), groupID.String())
		return err
	}))
	return result
}

func createGroupSettlementFixture(t *testing.T, database *sql.DB) groupSettlementFixture {
	t.Helper()

	fixture := groupSettlementFixture{
		actorID:                 uuid.New(),
		counterpartyID:          uuid.New(),
		nonPartyID:              uuid.New(),
		groupID:                 uuid.New(),
		unrelatedGroupID:        uuid.New(),
		openExpenseID:           uuid.New(),
		deletedExpenseID:        uuid.New(),
		unrelatedOpenExpenseID:  uuid.New(),
		openLedgerID:            uuid.New(),
		deletedLedgerID:         uuid.New(),
		unrelatedOpenLedgerID:   uuid.New(),
		currentBalanceID:        uuid.New(),
		historicalBalanceID:     uuid.New(),
		unrelatedCurrentBalance: uuid.New(),
	}
	expenseTypeID := uuid.New()
	now := time.Now().UTC()
	t.Cleanup(func() {
		expenseIDs := []uuid.UUID{fixture.openExpenseID, fixture.deletedExpenseID, fixture.unrelatedOpenExpenseID}
		_, cleanupErr := database.Exec("DELETE FROM ledger WHERE expense_id = ANY($1)", expenseIDs)
		require.NoError(t, cleanupErr)
		_, cleanupErr = database.Exec("DELETE FROM expense WHERE id = ANY($1)", expenseIDs)
		require.NoError(t, cleanupErr)
		_, cleanupErr = database.Exec("DELETE FROM groups WHERE id = ANY($1)", []uuid.UUID{fixture.groupID, fixture.unrelatedGroupID})
		require.NoError(t, cleanupErr)
		_, cleanupErr = database.Exec("DELETE FROM expense_type WHERE id = $1", expenseTypeID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = database.Exec("DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{fixture.actorID, fixture.counterpartyID, fixture.nonPartyID})
		require.NoError(t, cleanupErr)
	})

	for _, user := range []struct {
		id       uuid.UUID
		username string
	}{
		{id: fixture.actorID, username: "settlement-actor-" + fixture.actorID.String()[:8]},
		{id: fixture.counterpartyID, username: "settlement-counterparty-" + fixture.counterpartyID.String()[:8]},
		{id: fixture.nonPartyID, username: "settlement-nonparty-" + fixture.nonPartyID.String()[:8]},
	} {
		_, err := database.Exec(`INSERT INTO users (id, username, firstname, lastname, email, password_hash, create_time_utc, is_active, has_local_password, role)
			VALUES ($1, $2, 'Settlement', 'Test', $3, 'not-used', $4, TRUE, TRUE, 'user')`,
			user.id, user.username, user.username+"@example.test", now)
		require.NoError(t, err)
	}

	for _, group := range []struct {
		id   uuid.UUID
		name string
	}{
		{id: fixture.groupID, name: "settlement-group-" + fixture.groupID.String()[:8]},
		{id: fixture.unrelatedGroupID, name: "settlement-unrelated-" + fixture.unrelatedGroupID.String()[:8]},
	} {
		_, err := database.Exec(`INSERT INTO groups (id, group_name, description, create_time_utc, is_active, create_by_user_id, currency)
			VALUES ($1, $2, '', $3, TRUE, $4, 'CAD')`, group.id, group.name, now, fixture.actorID)
		require.NoError(t, err)
		for _, userID := range []uuid.UUID{fixture.actorID, fixture.counterpartyID, fixture.nonPartyID} {
			_, err = database.Exec(`INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)`, uuid.New(), group.id, userID)
			require.NoError(t, err)
		}
	}

	_, err := database.Exec(`INSERT INTO expense_type (id, name, category) VALUES ($1, $2, 'test')`,
		expenseTypeID, "settle-"+expenseTypeID.String()[:8])
	require.NoError(t, err)

	for _, expense := range []struct {
		id       uuid.UUID
		ledgerID uuid.UUID
		groupID  uuid.UUID
		deleted  bool
	}{
		{id: fixture.openExpenseID, ledgerID: fixture.openLedgerID, groupID: fixture.groupID},
		{id: fixture.deletedExpenseID, ledgerID: fixture.deletedLedgerID, groupID: fixture.groupID, deleted: true},
		{id: fixture.unrelatedOpenExpenseID, ledgerID: fixture.unrelatedOpenLedgerID, groupID: fixture.unrelatedGroupID},
	} {
		var deletedAt *time.Time
		if expense.deleted {
			deletedAt = &now
		}
		_, err = database.Exec(`INSERT INTO expense (
			id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id,
			is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc,
			allocation_mode, occurred_on, is_deleted, delete_time_utc
		) VALUES ($1, 'settlement characterization', $2, $3, $3, $4,
			FALSE, 10, 0, 10, 'CAD', $5, 'equal', $6, $7,
			$8)`,
			expense.id, expense.groupID, fixture.actorID, expenseTypeID, now, now.Format(time.DateOnly), expense.deleted, deletedAt)
		require.NoError(t, err)

		_, err = database.Exec(`INSERT INTO ledger (id, expense_id, lender_user_id, borrower_user_id, share)
			VALUES ($1, $2, $3, $4, 10)`, expense.ledgerID, expense.id, fixture.actorID, fixture.counterpartyID)
		require.NoError(t, err)
	}

	for _, balance := range []struct {
		id         uuid.UUID
		groupID    uuid.UUID
		isOutdated bool
	}{
		{id: fixture.currentBalanceID, groupID: fixture.groupID},
		{id: fixture.historicalBalanceID, groupID: fixture.groupID, isOutdated: true},
		{id: fixture.unrelatedCurrentBalance, groupID: fixture.unrelatedGroupID},
	} {
		_, err = database.Exec(`INSERT INTO balance (
			id, sender_user_id, receiver_user_id, share, group_id,
			create_time_utc, is_outdated, is_settled
		) VALUES ($1, $2, $3, 10, $4, $5, $6, FALSE)`,
			balance.id, fixture.counterpartyID, fixture.actorID, balance.groupID, now, balance.isOutdated)
		require.NoError(t, err)
	}

	for _, relationship := range []struct {
		balanceID uuid.UUID
		ledgerID  uuid.UUID
	}{
		{balanceID: fixture.currentBalanceID, ledgerID: fixture.openLedgerID},
		{balanceID: fixture.historicalBalanceID, ledgerID: fixture.openLedgerID},
		{balanceID: fixture.unrelatedCurrentBalance, ledgerID: fixture.unrelatedOpenLedgerID},
	} {
		_, err = database.Exec(`INSERT INTO balance_ledger (id, balance_id, ledger_id)
			VALUES ($1, $2, $3)`, uuid.New(), relationship.balanceID, relationship.ledgerID)
		require.NoError(t, err)
	}

	return fixture
}

func assertExpenseSettlementState(t *testing.T, database *sql.DB, expenseID uuid.UUID, wantSettled bool) {
	t.Helper()

	var settled bool
	var settledAt sql.NullTime
	require.NoError(t, database.QueryRow(
		"SELECT is_settled, settle_time_utc FROM expense WHERE id = $1", expenseID,
	).Scan(&settled, &settledAt))
	require.Equal(t, wantSettled, settled)
	require.Equal(t, wantSettled, settledAt.Valid)
}

func assertBalanceOutdatedState(t *testing.T, database *sql.DB, balanceID uuid.UUID, wantOutdated bool) {
	t.Helper()

	var outdated bool
	require.NoError(t, database.QueryRow(
		"SELECT is_outdated FROM balance WHERE id = $1", balanceID,
	).Scan(&outdated))
	require.Equal(t, wantOutdated, outdated)
}

func assertBalanceSettlementState(t *testing.T, database *sql.DB, balanceID uuid.UUID, wantSettled bool) {
	t.Helper()

	var settled bool
	var settledAt sql.NullTime
	require.NoError(t, database.QueryRow(
		"SELECT is_settled, settle_time_utc FROM balance WHERE id = $1", balanceID,
	).Scan(&settled, &settledAt))
	require.Equal(t, wantSettled, settled)
	require.Equal(t, wantSettled, settledAt.Valid)
}

func readExpenseSettlementTime(t *testing.T, database *sql.DB, expenseID uuid.UUID) time.Time {
	t.Helper()

	var settledAt time.Time
	require.NoError(t, database.QueryRow(
		"SELECT settle_time_utc FROM expense WHERE id = $1", expenseID,
	).Scan(&settledAt))
	return settledAt
}

func readBalanceUpdateTime(t *testing.T, database *sql.DB, balanceID uuid.UUID) time.Time {
	t.Helper()

	var updatedAt time.Time
	require.NoError(t, database.QueryRow(
		"SELECT update_time_utc FROM balance WHERE id = $1", balanceID,
	).Scan(&updatedAt))
	return updatedAt
}
