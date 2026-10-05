package group_test

import (
	"database/sql"
	expense "expense-tracker/backend/services/expense/stores"
	group "expense-tracker/backend/services/group/stores"
	"expense-tracker/backend/types"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func archiveFixture(t *testing.T, db *sql.DB) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	creator, member, id, category := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, ensureTestUser(db, creator))
	require.NoError(t, ensureTestUser(db, member))
	require.NoError(t, group.NewStore(db).CreateGroup(types.Group{ID: id, GroupName: "Archive fixture", Currency: "CAD", IsActive: true, CreateByUser: creator, CreateTime: time.Now().UTC(), GroupType: "home"}, []string{member.String()}))
	_, err := db.Exec(`INSERT INTO expense_type (id, name, category) VALUES ($1, 'Archive test', 'test')`, category)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM balance WHERE group_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM expense WHERE group_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM group_member WHERE group_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM groups WHERE id = $1`, id)
		_, _ = db.Exec(`DELETE FROM expense_type WHERE id = $1`, category)
		cleanUser(db, member)
		cleanUser(db, creator)
	})
	return id, creator, member, category
}

func TestArchiveLifecycleAccountingAndReadOnlyIntegration(t *testing.T) {
	db := openTestDB(t)
	id, creator, member, category := archiveFixture(t, db)
	store := group.NewStore(db)
	debt := uuid.New()
	_, err := db.Exec(`INSERT INTO expense (id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id, is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc, allocation_mode)
		VALUES ($1, 'Unsettled with no balance', $2, $3, $3, $4, FALSE, 10, 0, 10, 'CAD', NOW(), 'equal')`, debt, id, creator, category)
	require.NoError(t, err)
	require.ErrorIs(t, store.UpdateGroupStatus(id.String(), creator.String(), false), types.ErrGroupUnsettled)
	_, err = db.Exec(`UPDATE expense SET is_settled = TRUE WHERE id = $1`, debt)
	require.NoError(t, err)
	// An original-currency debt anywhere in the group blocks archive even when
	// the creator's displayed converted net could cancel to zero.
	_, err = db.Exec(`INSERT INTO balance (id, group_id, sender_user_id, receiver_user_id, share, currency, create_time_utc, update_time_utc, is_outdated, is_settled)
		VALUES ($1, $2, $3, $4, 10, 'CAD', NOW(), NOW(), FALSE, FALSE),
		($5, $2, $4, $3, 10, 'TWD', NOW(), NOW(), FALSE, FALSE)`, uuid.New(), id, creator, member, uuid.New())
	require.NoError(t, err)
	require.ErrorIs(t, store.UpdateGroupStatus(id.String(), creator.String(), false), types.ErrGroupUnsettled)
	_, err = db.Exec(`UPDATE balance SET is_settled = TRUE WHERE group_id = $1`, id)
	require.NoError(t, err)
	require.ErrorIs(t, store.UpdateGroupStatus(id.String(), member.String(), false), types.ErrGroupNotExist)
	require.NoError(t, store.UpdateGroupStatus(id.String(), creator.String(), false))
	require.NoError(t, store.UpdateGroupStatus(id.String(), creator.String(), false))
	state, err := store.GetGroupLifecycle(id.String(), creator.String())
	require.NoError(t, err)
	require.True(t, state.CanRestore)
	require.False(t, state.CanArchive)
	active, err := store.GetGroupListByUser(creator.String())
	require.NoError(t, err)
	for _, card := range active {
		require.NotEqual(t, id.String(), card.ID)
	}
	archived, err := store.GetArchivedGroups(creator.String(), "", 100)
	require.NoError(t, err)
	found := false
	for _, card := range archived.Groups {
		if card.ID == id.String() {
			found = true
		}
	}
	require.True(t, found)
	require.ErrorIs(t, store.UpdateGroup(types.Group{ID: id, CreateByUser: creator, GroupName: "Forbidden", GroupType: "home"}), types.ErrGroupNotExist)
	require.ErrorIs(t, store.ReplaceGroupMembers(id.String(), creator.String(), nil), types.ErrGroupNotExist)
	require.ErrorIs(t, store.UpdateGroupCurrency(id.String(), creator.String(), "CAD"), types.ErrGroupNotExist)
	require.ErrorIs(t, expense.NewStore(db).RunInTransaction(func(tx types.ExpenseTransactionStore) error { _, err := tx.LockGroupCurrency(id.String()); return err }), types.ErrGroupNotExist)
	_, err = store.GetGroupByIDAndUser(id.String(), member.String())
	require.NoError(t, err)
	require.NoError(t, store.UpdateGroupStatus(id.String(), creator.String(), true))
	require.NoError(t, store.UpdateGroupStatus(id.String(), creator.String(), true))
}

func TestArchiveWaitsForConcurrentAccountingMutationIntegration(t *testing.T) {
	db := openTestDB(t)
	id, creator, _, category := archiveFixture(t, db)
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	var locked uuid.UUID
	require.NoError(t, tx.QueryRow(`SELECT id FROM groups WHERE id = $1 AND is_active = TRUE FOR UPDATE`, id).Scan(&locked))
	result := make(chan error, 1)
	go func() { result <- group.NewStore(db).UpdateGroupStatus(id.String(), creator.String(), false) }()
	select {
	case err := <-result:
		t.Fatalf("archive completed while accounting held the group lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	_, err = tx.Exec(`INSERT INTO expense (id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id, is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc, allocation_mode)
		VALUES ($1, 'Concurrent expense', $2, $3, $3, $4, FALSE, 10, 0, 10, 'CAD', NOW(), 'equal')`, uuid.New(), id, creator, category)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	select {
	case err := <-result:
		require.ErrorIs(t, err, types.ErrGroupUnsettled)
	case <-time.After(5 * time.Second):
		t.Fatal("archive did not resume after accounting committed")
	}
}

func TestArchivedPaginationTiesMembershipAndLifecycleChangesIntegration(t *testing.T) {
	db := openTestDB(t)
	_, creator, outsider, _ := archiveFixture(t, db)
	store := group.NewStore(db)
	created := time.Now().UTC().Truncate(time.Second)
	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	slices.SortFunc(ids, func(a, b string) int { return strings.Compare(b, a) })
	for _, id := range append(slices.Clone(ids), uuid.NewString()) {
		owner := creator
		if !slices.Contains(ids, id) {
			owner = outsider
		}
		require.NoError(t, store.CreateGroup(types.Group{ID: uuid.MustParse(id), GroupName: "Pagination fixture", Currency: "CAD", GroupType: "home", IsActive: false, CreateByUser: owner, CreateTime: created}, nil))
		t.Cleanup(func() {
			_, _ = db.Exec(`DELETE FROM group_member WHERE group_id = $1`, id)
			_, _ = db.Exec(`DELETE FROM groups WHERE id = $1`, id)
		})
	}
	first, err := store.GetArchivedGroups(creator.String(), "", 1)
	require.NoError(t, err)
	require.Len(t, first.Groups, 1)
	require.Equal(t, ids[0], first.Groups[0].ID)
	require.NotEmpty(t, first.NextCursor)
	// Moving the preceding card back to active does not reset the cursor or
	// expose an archived group owned by an unrelated member.
	require.NoError(t, store.UpdateGroupStatus(ids[0], creator.String(), true))
	second, err := store.GetArchivedGroups(creator.String(), first.NextCursor, 1)
	require.NoError(t, err)
	require.Len(t, second.Groups, 1)
	require.Equal(t, ids[1], second.Groups[0].ID)
	third, err := store.GetArchivedGroups(creator.String(), second.NextCursor, 1)
	require.NoError(t, err)
	require.Len(t, third.Groups, 1)
	require.Equal(t, ids[2], third.Groups[0].ID)
	require.Empty(t, third.NextCursor)
	outsiderPage, err := store.GetArchivedGroups(outsider.String(), "", 100)
	require.NoError(t, err)
	require.Len(t, outsiderPage.Groups, 1)
	require.NotContains(t, ids, outsiderPage.Groups[0].ID)
}

func TestHomeArchiveSuggestionEligibilityIntegration(t *testing.T) {
	db := openTestDB(t)
	id, creator, member, category := archiveFixture(t, db)
	store := group.NewStore(db)
	_, err := db.Exec(`UPDATE groups SET group_type = 'trip', create_time_utc = NOW() - INTERVAL '130 days' WHERE id = $1`, id)
	require.NoError(t, err)
	expenseID := uuid.New()
	_, err = db.Exec(`INSERT INTO expense (id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id, is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc, update_time_utc, settle_time_utc, allocation_mode)
		VALUES ($1, 'Completed trip', $2, $3, $3, $4, TRUE, 10, 0, 10, 'CAD', NOW() - INTERVAL '120 days', NOW() - INTERVAL '120 days', NOW() - INTERVAL '120 days', 'equal')`, expenseID, id, creator, category)
	require.NoError(t, err)
	check := func(actor uuid.UUID, want bool) {
		t.Helper()
		cards, err := store.GetGroupListByUser(actor.String())
		require.NoError(t, err)
		for _, card := range cards {
			if card.ID == id.String() {
				require.Equal(t, want, card.ArchiveSuggested)
				return
			}
		}
		t.Fatal("expected active fixture in group list")
	}
	check(creator, true)
	check(member, false)
	for _, kind := range []string{"home", "family", "event", "trip"} {
		_, err = db.Exec(`UPDATE groups SET group_type = $1 WHERE id = $2`, kind, id)
		require.NoError(t, err)
		check(creator, kind == "trip" || kind == "event")
	}
	_, err = db.Exec(`UPDATE expense SET is_settled = FALSE WHERE id = $1`, expenseID)
	require.NoError(t, err)
	check(creator, false)
	_, err = db.Exec(`UPDATE expense SET is_settled = TRUE, update_time_utc = NOW() WHERE id = $1`, expenseID)
	require.NoError(t, err)
	check(creator, false) // A backdated expense with a recent edit is still recent activity.
	_, err = db.Exec(`UPDATE expense SET update_time_utc = NOW() - INTERVAL '120 days', is_deleted = TRUE, delete_time_utc = NOW() WHERE id = $1`, expenseID)
	require.NoError(t, err)
	check(creator, false)
	_, err = db.Exec(`UPDATE expense SET is_deleted = FALSE, delete_time_utc = NULL WHERE id = $1`, expenseID)
	require.NoError(t, err)
	check(creator, true)
	balanceID := uuid.New()
	thirdMember := uuid.New()
	require.NoError(t, ensureTestUser(db, thirdMember))
	require.NoError(t, store.UpdateGroupMember("add", thirdMember.String(), id.String()))
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM balance WHERE group_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM group_member WHERE group_id = $1 AND user_id = $2`, id, thirdMember)
		cleanUser(db, thirdMember)
	})
	_, err = db.Exec(`INSERT INTO balance (id, group_id, sender_user_id, receiver_user_id, share, currency, is_outdated, is_settled, settle_time_utc)
		VALUES ($1, $2, $3, $4, 10, 'TWD', FALSE, FALSE, NULL)`, balanceID, id, member, thirdMember)
	require.NoError(t, err)
	check(creator, false)
	_, err = db.Exec(`UPDATE balance SET is_settled = TRUE, settle_time_utc = NOW() WHERE id = $1`, balanceID)
	require.NoError(t, err)
	check(creator, false)
	_, err = db.Exec(`UPDATE balance SET settle_time_utc = NOW() - INTERVAL '120 days' WHERE id = $1`, balanceID)
	require.NoError(t, err)
	check(creator, true)
	_, err = db.Exec(`UPDATE expense SET settle_time_utc = NOW() - INTERVAL '90 days' WHERE id = $1`, expenseID)
	require.NoError(t, err)
	check(creator, true)
	_, err = db.Exec(`UPDATE expense SET settle_time_utc = NOW() - INTERVAL '89 days' WHERE id = $1`, expenseID)
	require.NoError(t, err)
	check(creator, false)
}
