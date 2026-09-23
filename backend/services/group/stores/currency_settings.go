package group

import (
	"database/sql"
	"errors"
	"strings"

	"expense-tracker/backend/types"

	"github.com/shopspring/decimal"
)

func (s *Store) GetGroupCurrencySettings(groupID string, userID string) (types.GroupCurrencySettings, error) {
	var previewCurrency string
	err := s.db.QueryRow(`SELECT btrim(groups.settlement_preview_currency)
		FROM groups
		JOIN group_member ON group_member.group_id = groups.id
		WHERE groups.id = $1 AND group_member.user_id = $2`, groupID, userID).Scan(&previewCurrency)
	if errors.Is(err, sql.ErrNoRows) {
		return types.GroupCurrencySettings{}, types.ErrGroupNotExist
	}
	if err != nil {
		return types.GroupCurrencySettings{}, err
	}

	rows, err := s.db.Query(`SELECT
		btrim(group_currency.currency),
		group_currency.enabled_for_new_expenses,
		group_currency.preview_rate,
		EXISTS (
			SELECT 1 FROM expense
			WHERE expense.group_id = group_currency.group_id
			  AND expense.currency = group_currency.currency
		),
		EXISTS (
			SELECT 1 FROM balance
			WHERE balance.group_id = group_currency.group_id
			  AND balance.currency = group_currency.currency
			  AND balance.is_outdated = FALSE
			  AND balance.is_settled = FALSE
		)
	FROM group_currency
	WHERE group_currency.group_id = $1
	ORDER BY group_currency.currency`, groupID)
	if err != nil {
		return types.GroupCurrencySettings{}, err
	}
	defer rows.Close()

	settings := types.GroupCurrencySettings{
		SettlementPreviewCurrency: previewCurrency,
		Currencies:                make([]types.GroupCurrencySetting, 0),
	}
	for rows.Next() {
		var item types.GroupCurrencySetting
		var rate decimal.NullDecimal
		if err := rows.Scan(
			&item.Currency,
			&item.EnabledForNewExpenses,
			&rate,
			&item.Historical,
			&item.HasCurrentBalance,
		); err != nil {
			return types.GroupCurrencySettings{}, err
		}
		if rate.Valid {
			value := rate.Decimal
			item.PreviewRate = &value
		}
		settings.Currencies = append(settings.Currencies, item)
	}
	if err := rows.Err(); err != nil {
		return types.GroupCurrencySettings{}, err
	}
	return settings, nil
}

func (s *Store) UpdateGroupCurrencySettings(groupID string, userID string, settings types.GroupCurrencySettings) error {
	previewCurrency := strings.ToUpper(strings.TrimSpace(settings.SettlementPreviewCurrency))
	if previewCurrency == "" || len(settings.Currencies) == 0 {
		return types.ErrInvalidCurrencySettings
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var lockedID string
	err = tx.QueryRow(`SELECT groups.id
		FROM groups
		JOIN group_member ON group_member.group_id = groups.id
		WHERE groups.id = $1 AND group_member.user_id = $2 AND groups.is_active = TRUE
		FOR UPDATE OF groups`, groupID, userID).Scan(&lockedID)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrGroupNotExist
	}
	if err != nil {
		return err
	}

	incoming := make(map[string]types.GroupCurrencySetting, len(settings.Currencies))
	enabledCount := 0
	for _, item := range settings.Currencies {
		code := strings.ToUpper(strings.TrimSpace(item.Currency))
		if code == "" || code != item.Currency {
			return types.ErrInvalidCurrencySettings
		}
		if _, duplicate := incoming[code]; duplicate {
			return types.ErrInvalidCurrencySettings
		}
		var supported bool
		if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM currency WHERE code = $1)`, code).Scan(&supported); err != nil {
			return err
		}
		if !supported || (item.PreviewRate != nil && !item.PreviewRate.IsPositive()) {
			return types.ErrInvalidCurrencySettings
		}
		if code == previewCurrency {
			one := decimal.NewFromInt(1)
			item.PreviewRate = &one
		}
		item.Currency = code
		incoming[code] = item
		if item.EnabledForNewExpenses {
			enabledCount++
		}
	}
	preview, hasPreview := incoming[previewCurrency]
	if !hasPreview || !preview.EnabledForNewExpenses || enabledCount == 0 {
		return types.ErrInvalidCurrencySettings
	}

	requiredRows, err := tx.Query(`SELECT btrim(group_currency.currency)
		FROM group_currency
		WHERE group_currency.group_id = $1
		  AND (
			EXISTS (SELECT 1 FROM expense WHERE expense.group_id = $1 AND expense.currency = group_currency.currency)
			OR EXISTS (
				SELECT 1 FROM balance
				WHERE balance.group_id = $1
				  AND balance.currency = group_currency.currency
				  AND balance.is_outdated = FALSE
				  AND balance.is_settled = FALSE
			)
		  )`, groupID)
	if err != nil {
		return err
	}
	for requiredRows.Next() {
		var code string
		if err := requiredRows.Scan(&code); err != nil {
			requiredRows.Close()
			return err
		}
		if _, ok := incoming[code]; !ok {
			requiredRows.Close()
			return types.ErrInvalidCurrencySettings
		}
	}
	if err := requiredRows.Close(); err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE groups SET settlement_preview_currency = $1 WHERE id = $2`, previewCurrency, groupID); err != nil {
		return err
	}
	for code, item := range incoming {
		var rate any
		if item.PreviewRate != nil {
			rate = item.PreviewRate.String()
		}
		if _, err := tx.Exec(`INSERT INTO group_currency (
			group_id, currency, enabled_for_new_expenses, preview_rate
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (group_id, currency) DO UPDATE SET
			enabled_for_new_expenses = EXCLUDED.enabled_for_new_expenses,
			preview_rate = EXCLUDED.preview_rate`, groupID, code, item.EnabledForNewExpenses, rate); err != nil {
			return err
		}
	}

	existingRows, err := tx.Query(`SELECT btrim(currency) FROM group_currency WHERE group_id = $1`, groupID)
	if err != nil {
		return err
	}
	var removable []string
	for existingRows.Next() {
		var code string
		if err := existingRows.Scan(&code); err != nil {
			existingRows.Close()
			return err
		}
		if _, keep := incoming[code]; !keep {
			removable = append(removable, code)
		}
	}
	if err := existingRows.Close(); err != nil {
		return err
	}
	for _, code := range removable {
		result, err := tx.Exec(`DELETE FROM group_currency
			WHERE group_id = $1 AND currency = $2
			  AND NOT EXISTS (SELECT 1 FROM expense WHERE group_id = $1 AND currency = $2)
			  AND NOT EXISTS (
				SELECT 1 FROM balance
				WHERE group_id = $1 AND currency = $2
				  AND is_outdated = FALSE AND is_settled = FALSE
			  )`, groupID, code)
		if err != nil {
			return err
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if deleted == 0 {
			return types.ErrInvalidCurrencySettings
		}
	}
	return tx.Commit()
}
