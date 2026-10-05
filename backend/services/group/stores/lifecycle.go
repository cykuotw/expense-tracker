package group

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"expense-tracker/backend/types"
	"time"

	"github.com/google/uuid"
)

// Checks original accounting rows, never a user's net or converted preview.
const groupSettledSQL = `SELECT
	NOT EXISTS (SELECT 1 FROM expense WHERE group_id = $1 AND is_deleted = FALSE AND is_settled = FALSE)
	AND NOT EXISTS (SELECT 1 FROM balance WHERE group_id = $1 AND is_outdated = FALSE AND is_settled = FALSE)`

func (s *Store) GetGroupLifecycle(groupID, userID string) (types.GroupLifecycle, error) {
	var state types.GroupLifecycle
	var creator, settled bool
	err := s.db.QueryRow(`SELECT g.is_active, g.create_by_user_id = $2,
		NOT EXISTS (SELECT 1 FROM expense WHERE group_id = g.id AND is_deleted = FALSE AND is_settled = FALSE)
		AND NOT EXISTS (SELECT 1 FROM balance WHERE group_id = g.id AND is_outdated = FALSE AND is_settled = FALSE)
		FROM groups g JOIN group_member gm ON gm.group_id = g.id
		WHERE g.id = $1 AND gm.user_id = $2`, groupID, userID).Scan(&state.IsActive, &creator, &settled)
	if errors.Is(err, sql.ErrNoRows) {
		return state, types.ErrGroupNotExist
	}
	if err != nil {
		return state, err
	}
	state.CanManageLifecycle = creator
	state.CanArchive = creator && state.IsActive && settled
	state.CanRestore = creator && !state.IsActive
	if state.IsActive && !settled {
		state.ArchiveBlockedReason = types.ErrGroupUnsettled.Error()
	}
	return state, nil
}

type archivedCursor struct {
	Created time.Time `json:"created"`
	ID      uuid.UUID `json:"id"`
}

func decodeArchivedCursor(value string) (archivedCursor, error) {
	var cursor archivedCursor
	if len(value) > 512 {
		return cursor, types.ErrInvalidGroupCursor
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || json.Unmarshal(data, &cursor) != nil || cursor.Created.IsZero() || cursor.ID == uuid.Nil {
		return cursor, types.ErrInvalidGroupCursor
	}
	return cursor, nil
}

func (s *Store) GetArchivedGroups(userID, value string, limit int) (types.ArchivedGroupPage, error) {
	page := types.ArchivedGroupPage{Groups: make([]types.GetGroupListResponse, 0)}
	if limit < 1 || limit > 100 {
		return page, types.ErrInvalidGroupCursor
	}
	var cursor archivedCursor
	if value != "" {
		var err error
		cursor, err = decodeArchivedCursor(value)
		if err != nil {
			return page, err
		}
	}
	rows, err := s.db.Query(`SELECT g.id, g.group_name, g.description,
		btrim(g.settlement_preview_currency), g.group_type, g.create_time_utc
		FROM groups g JOIN group_member gm ON gm.group_id = g.id
		WHERE gm.user_id = $1 AND g.is_active = FALSE
		AND ($2 = '' OR (g.create_time_utc, g.id) < ($3, $4))
		ORDER BY g.create_time_utc DESC, g.id DESC LIMIT $5`, userID, value, cursor.Created, cursor.ID, limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	var last archivedCursor
	for rows.Next() {
		var card types.GetGroupListResponse
		var position archivedCursor
		if err := rows.Scan(&card.ID, &card.GroupName, &card.Description, &card.Currency, &card.GroupType, &position.Created); err != nil {
			return page, err
		}
		if len(page.Groups) == limit {
			data, err := json.Marshal(last)
			if err != nil {
				return page, err
			}
			page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
			break
		}
		position.ID, err = uuid.Parse(card.ID)
		if err != nil {
			return page, err
		}
		card.BalanceStatus = types.GroupBalanceStatusSettled
		card.SettlementPreviewComplete = true
		page.Groups = append(page.Groups, card)
		last = position
	}
	return page, rows.Err()
}
