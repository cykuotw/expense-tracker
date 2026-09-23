package group

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func (s *Store) CreateGroup(group types.Group, memberIDs []string) error {
	if group.GroupType == "" {
		group.GroupType = "home"
	}
	if group.SettlementPreviewCurrency == "" {
		group.SettlementPreviewCurrency = group.Currency
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	createTime := group.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := "INSERT INTO groups (" +
		"id, group_name, description, " +
		"create_time_utc, is_active, currency, settlement_preview_currency, create_by_user_id, group_type" +
		") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);"

	_, err = tx.Exec(query,
		group.ID, group.GroupName, group.Description,
		createTime, group.IsActive, group.Currency, group.SettlementPreviewCurrency, group.CreateByUser, group.GroupType)
	if err != nil {
		return err
	}

	for _, setting := range group.CurrencySettings.Currencies {
		var rate any
		if setting.PreviewRate != nil {
			rate = setting.PreviewRate.String()
		}
		if setting.Currency == group.SettlementPreviewCurrency {
			rate = "1"
		}
		if _, err := tx.Exec(`INSERT INTO group_currency (
			group_id, currency, enabled_for_new_expenses, preview_rate
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (group_id, currency) DO UPDATE SET
			enabled_for_new_expenses = EXCLUDED.enabled_for_new_expenses,
			preview_rate = EXCLUDED.preview_rate`,
			group.ID, setting.Currency, setting.EnabledForNewExpenses, rate); err != nil {
			return err
		}
	}

	memberSet := make(map[string]struct{}, len(memberIDs)+1)
	memberSet[group.CreateByUser.String()] = struct{}{}
	for _, memberID := range memberIDs {
		memberSet[memberID] = struct{}{}
	}
	for memberID := range memberSet {
		if _, err := tx.Exec(
			"INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)",
			uuid.NewString(), group.ID, memberID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
