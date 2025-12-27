package group

import (
	"database/sql"
	"expense-tracker/backend/types"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func scanRowIntoGroup(rows *sql.Rows) (*types.Group, error) {
	group := new(types.Group)

	err := rows.Scan(
		&group.ID,
		&group.GroupName,
		&group.Description,
		&group.CreateTime,
		&group.IsActive,
		&group.CreateByUser,
		&group.Currency,
	)
	if err != nil {
		return nil, err
	}
	return group, nil
}
