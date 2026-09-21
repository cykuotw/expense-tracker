package group

import (
	"database/sql"

	"expense-tracker/backend/types"
)

type Store struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...any) error
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func scanRowIntoGroup(row rowScanner) (*types.Group, error) {
	group := new(types.Group)

	err := row.Scan(
		&group.ID,
		&group.GroupName,
		&group.Description,
		&group.CreateTime,
		&group.IsActive,
		&group.CreateByUser,
		&group.Currency,
		&group.GroupType,
	)
	if err != nil {
		return nil, err
	}
	return group, nil
}
