package group

import (
	"expense-tracker/backend/types"

	"github.com/shopspring/decimal"
)

func (s *Store) GetGroupListByUser(userID string) ([]types.GetGroupListResponse, error) {
	query := `
		WITH member_groups AS (
			SELECT
				g.id,
				g.group_name,
				g.description,
				btrim(g.settlement_preview_currency) AS currency,
				c.minor_unit_digits,
				g.group_type,
				g.create_time_utc
			FROM groups g
			INNER JOIN group_member gm ON gm.group_id = g.id
			INNER JOIN currency c ON c.code = g.settlement_preview_currency
			WHERE gm.user_id = $1
		),
		balance_net AS (
			SELECT
				b.group_id,
				COUNT(b.id) AS balance_count,
				COUNT(DISTINCT b.currency) AS currency_count,
				MIN(b.currency) AS balance_currency,
				COALESCE(BOOL_OR(gc.preview_rate IS NULL), FALSE) AS missing_rate,
				COALESCE(SUM(
					CASE
						WHEN b.receiver_user_id = $1 THEN b.share
						WHEN b.sender_user_id = $1 THEN -b.share
						ELSE 0
					END
				), 0) AS original_net,
				COALESCE(SUM(ROUND(
					CASE
						WHEN b.receiver_user_id = $1 THEN b.share * gc.preview_rate
						WHEN b.sender_user_id = $1 THEN -b.share * gc.preview_rate
						ELSE 0
					END,
					mg.minor_unit_digits
				)), 0) AS preview_net
			FROM balance b
			INNER JOIN member_groups mg ON mg.id = b.group_id
			LEFT JOIN group_currency gc ON gc.group_id = b.group_id AND gc.currency = b.currency
			WHERE b.is_outdated = FALSE AND b.is_settled = FALSE
			  AND (b.receiver_user_id = $1 OR b.sender_user_id = $1)
			GROUP BY b.group_id
		),
		expense_activity AS (
			SELECT
				e.group_id,
				MAX(e.update_time_utc) AS last_activity
			FROM expense e
			INNER JOIN member_groups mg ON mg.id = e.group_id
			WHERE e.is_deleted = FALSE
			GROUP BY e.group_id
		)
		SELECT
			mg.id,
			mg.group_name,
			mg.description,
			CASE
				WHEN COALESCE(bn.currency_count, 0) = 1 THEN bn.balance_currency
				ELSE mg.currency
			END AS currency,
			mg.group_type,
			CASE
				WHEN COALESCE(bn.balance_count, 0) = 0 THEN 'settled'
				WHEN bn.currency_count > 1 AND bn.missing_rate THEN 'preview_unavailable'
				WHEN CASE
					WHEN bn.currency_count > 1 THEN COALESCE(bn.preview_net, 0)
					ELSE COALESCE(bn.original_net, 0)
				END = 0 THEN 'settled'
				WHEN CASE
					WHEN bn.currency_count > 1 THEN COALESCE(bn.preview_net, 0)
					ELSE COALESCE(bn.original_net, 0)
				END > 0 THEN 'owed'
				ELSE 'owing'
			END AS balance_status,
			ABS(CASE
				WHEN bn.currency_count > 1 THEN COALESCE(bn.preview_net, 0)
				ELSE COALESCE(bn.original_net, 0)
			END) AS balance_amount,
			NOT (
				COALESCE(bn.currency_count, 0) > 1
				AND COALESCE(bn.missing_rate, FALSE)
			) AS settlement_preview_complete,
			COALESCE(bn.currency_count, 0) > 1 AS uses_settlement_preview
		FROM member_groups mg
		LEFT JOIN balance_net bn ON bn.group_id = mg.id
		LEFT JOIN expense_activity ea ON ea.group_id = mg.id
		ORDER BY
			CASE WHEN CASE
				WHEN bn.currency_count > 1 THEN COALESCE(bn.preview_net, 0)
				ELSE COALESCE(bn.original_net, 0)
			END = 0 THEN 1 ELSE 0 END ASC,
			COALESCE(ea.last_activity, mg.create_time_utc) DESC,
			mg.create_time_utc DESC;
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]types.GetGroupListResponse, 0)
	for rows.Next() {
		var group types.GetGroupListResponse
		var status string
		var amount decimal.Decimal

		if err := rows.Scan(
			&group.ID,
			&group.GroupName,
			&group.Description,
			&group.Currency,
			&group.GroupType,
			&status,
			&amount,
			&group.SettlementPreviewComplete,
			&group.UsesSettlementPreview,
		); err != nil {
			return nil, err
		}

		group.BalanceStatus = types.GroupBalanceStatus(status)
		group.BalanceAmount = amount
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}
