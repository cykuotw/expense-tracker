package expense

import (
	"testing"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUpdateBalancePartitionsLedgersByCurrency(t *testing.T) {
	borrower := uuid.New()
	lender := uuid.New()
	cadLedger := &types.Ledger{ID: uuid.New(), LenderUserID: lender, BorrowerUesrID: borrower, Share: decimal.NewFromInt(10), Currency: "CAD"}
	jpyLedger := &types.Ledger{ID: uuid.New(), LenderUserID: lender, BorrowerUesrID: borrower, Share: decimal.NewFromInt(900), Currency: "JPY"}
	store := expenseStoreMock()
	store.GetLedgerUnsettledFromGroupFn = func(string) ([]*types.Ledger, error) {
		return []*types.Ledger{jpyLedger, cadLedger}, nil
	}
	var createdCurrencies []string
	store.CreateBalancesFn = func(_ string, balances []*types.Balance) error {
		require.Len(t, balances, 1)
		createdCurrencies = append(createdCurrencies, balances[0].Currency)
		return nil
	}
	var linkedLedgerIDs [][]uuid.UUID
	store.CreateBalanceLedgerFn = func(_ []uuid.UUID, ledgerIDs []uuid.UUID) error {
		linkedLedgerIDs = append(linkedLedgerIDs, ledgerIDs)
		return nil
	}

	controller := expenseControllerMock()
	controller.DebtSimplifyFn = func(ledgers []*types.Ledger) []*types.Balance {
		require.Len(t, ledgers, 1)
		return []*types.Balance{{
			SenderUserID: ledgers[0].BorrowerUesrID, ReceiverUserID: ledgers[0].LenderUserID,
			Share: ledgers[0].Share, Currency: ledgers[0].Currency,
		}}
	}
	handler := &Handler{controller: controller}
	require.NoError(t, handler.updateBalanceWithStore(store, mockGroupID.String()))
	require.Equal(t, []string{"CAD", "JPY"}, createdCurrencies)
	require.Equal(t, [][]uuid.UUID{{cadLedger.ID}, {jpyLedger.ID}}, linkedLedgerIDs)
}

type groupSettingsMock struct {
	*mockGroupStore
	settings types.GroupCurrencySettings
}

func (m *groupSettingsMock) GetGroupCurrencySettings(string, string) (types.GroupCurrencySettings, error) {
	return m.settings, nil
}

func (m *groupSettingsMock) UpdateGroupCurrencySettings(string, string, types.GroupCurrencySettings) error {
	return nil
}

func TestSettlementPreviewRoundsEachBalanceBeforeAggregating(t *testing.T) {
	userID := uuid.New()
	rate := decimal.RequireFromString("0.004")
	one := decimal.NewFromInt(1)
	groupStore := &groupSettingsMock{
		mockGroupStore: groupStoreMock(),
		settings: types.GroupCurrencySettings{
			SettlementPreviewCurrency: "CAD",
			Currencies: []types.GroupCurrencySetting{
				{Currency: "CAD", PreviewRate: &one},
				{Currency: "TWD", PreviewRate: &rate},
			},
		},
	}
	groupStore.ListCurrenciesFn = func() ([]types.Currency, error) {
		return []types.Currency{{Code: "CAD", MinorUnitDigits: 2}}, nil
	}
	balances := []types.Balance{
		{ID: uuid.New(), ReceiverUserID: userID, Share: decimal.NewFromInt(10), Currency: "CAD"},
		{ID: uuid.New(), ReceiverUserID: userID, Share: decimal.NewFromInt(1), Currency: "TWD"},
		{ID: uuid.New(), ReceiverUserID: userID, Share: decimal.NewFromInt(1), Currency: "TWD"},
	}

	preview, err := (&Handler{groupStore: groupStore}).settlementPreview(
		mockGroupID.String(), userID.String(), balances, "CAD",
	)
	require.NoError(t, err)
	require.True(t, preview.Complete)
	require.NotNil(t, preview.NetAmount)
	require.True(t, preview.NetAmount.Equal(decimal.NewFromInt(10)))
	require.True(t, preview.Contributions[1].PreviewAmount.IsZero())
	require.True(t, preview.Contributions[2].PreviewAmount.IsZero())
}

func TestSettlementPreviewIsIncompleteWhenRateIsMissing(t *testing.T) {
	userID := uuid.New()
	one := decimal.NewFromInt(1)
	groupStore := &groupSettingsMock{
		mockGroupStore: groupStoreMock(),
		settings: types.GroupCurrencySettings{
			SettlementPreviewCurrency: "CAD",
			Currencies: []types.GroupCurrencySetting{
				{Currency: "CAD", PreviewRate: &one},
				{Currency: "JPY"},
			},
		},
	}
	groupStore.ListCurrenciesFn = func() ([]types.Currency, error) {
		return []types.Currency{{Code: "CAD", MinorUnitDigits: 2}}, nil
	}

	preview, err := (&Handler{groupStore: groupStore}).settlementPreview(
		mockGroupID.String(), userID.String(), []types.Balance{{
			ID: uuid.New(), ReceiverUserID: userID, Share: decimal.NewFromInt(100), Currency: "JPY",
		}}, "CAD",
	)
	require.NoError(t, err)
	require.False(t, preview.Complete)
	require.Nil(t, preview.NetAmount)
	require.Equal(t, []string{"JPY"}, preview.MissingRateCurrencies)
	require.Nil(t, preview.Contributions[0].PreviewAmount)
}
