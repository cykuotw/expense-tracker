package group

import (
	"database/sql"
	"errors"
	"strings"

	"expense-tracker/backend/types"
)

func (s *Store) ListCurrencies() ([]types.Currency, error) {
	rows, err := s.db.Query(`SELECT btrim(code), display_name, minor_unit_digits, amount_digits
		FROM currency
		ORDER BY CASE btrim(code)
			WHEN 'CAD' THEN 0
			WHEN 'TWD' THEN 1
			WHEN 'USD' THEN 2
			ELSE 3
		END, btrim(code) ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	currencies := make([]types.Currency, 0)
	for rows.Next() {
		var currency types.Currency
		if err := rows.Scan(&currency.Code, &currency.DisplayName, &currency.MinorUnitDigits, &currency.AmountDigits); err != nil {
			return nil, err
		}
		currencies = append(currencies, currency)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return currencies, nil
}

func (s *Store) IsSupportedCurrency(code string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM currency WHERE code = $1)`, code).Scan(&exists)
	return exists, err
}

func (s *Store) CanEditGroupCurrency(groupID string, userID string) (bool, error) {
	var editable bool
	err := s.db.QueryRow(`SELECT NOT EXISTS (
		SELECT 1 FROM expense WHERE expense.group_id = groups.id
	)
	FROM groups
	JOIN group_member ON group_member.group_id = groups.id
	WHERE groups.id = $1 AND group_member.user_id = $2`, groupID, userID).Scan(&editable)
	if errors.Is(err, sql.ErrNoRows) {
		return false, types.ErrGroupNotExist
	}
	return editable, err
}

func (s *Store) UpdateGroupCurrency(groupID string, userID string, currency string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var current string
	err = tx.QueryRow(`SELECT groups.currency
		FROM groups
		JOIN group_member ON group_member.group_id = groups.id
		WHERE groups.id = $1 AND group_member.user_id = $2
		FOR UPDATE OF groups`, groupID, userID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrGroupNotExist
	}
	if err != nil {
		return err
	}

	if currency != strings.ToUpper(currency) {
		return types.ErrUnsupportedCurrency
	}
	var supported bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM currency WHERE code = $1)`, currency).Scan(&supported); err != nil {
		return err
	}
	if !supported {
		return types.ErrUnsupportedCurrency
	}

	var hasExpense bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM expense WHERE group_id = $1)`, groupID).Scan(&hasExpense); err != nil {
		return err
	}
	if hasExpense && current != currency {
		return types.ErrGroupCurrencyLocked
	}
	if current == currency {
		return tx.Commit()
	}

	if _, err := tx.Exec(`UPDATE groups SET currency = $1 WHERE id = $2`, currency, groupID); err != nil {
		return err
	}
	return tx.Commit()
}
