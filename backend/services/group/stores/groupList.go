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
				g.currency,
				g.create_time_utc
			FROM groups g
			INNER JOIN group_member gm ON gm.group_id = g.id
			WHERE gm.user_id = $1
		),
		balance_net AS (
			SELECT
				b.group_id,
				COALESCE(SUM(
					CASE
						WHEN b.receiver_user_id = $1 THEN b.share
						WHEN b.sender_user_id = $1 THEN -b.share
						ELSE 0
					END
				), 0) AS net
			FROM balance b
			INNER JOIN member_groups mg ON mg.id = b.group_id
			WHERE b.is_outdated = FALSE AND b.is_settled = FALSE
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
			mg.currency,
			CASE
				WHEN COALESCE(bn.net, 0) = 0 THEN 'settled'
				WHEN COALESCE(bn.net, 0) > 0 THEN 'owed'
				ELSE 'owing'
			END AS balance_status,
			ABS(COALESCE(bn.net, 0)) AS balance_amount
		FROM member_groups mg
		LEFT JOIN balance_net bn ON bn.group_id = mg.id
		LEFT JOIN expense_activity ea ON ea.group_id = mg.id
		ORDER BY
			CASE WHEN COALESCE(bn.net, 0) = 0 THEN 1 ELSE 0 END ASC,
			COALESCE(ea.last_activity, mg.create_time_utc) DESC,
			mg.create_time_utc DESC;
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []types.GetGroupListResponse
	for rows.Next() {
		var group types.GetGroupListResponse
		var status string
		var amount decimal.Decimal

		if err := rows.Scan(
			&group.ID,
			&group.GroupName,
			&group.Description,
			&group.Currency,
			&status,
			&amount,
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

	if len(groups) == 0 {
		return nil, nil
	}

	return groups, nil
}
