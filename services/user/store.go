package user

import (
	"database/sql"
	"expense-tracker/types"
	"fmt"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetUserByEmail(email string) (*types.User, error) {
	statement := fmt.Sprintf("SELECT * FROM users WHERE email = '%s';", email)
	rows, err := s.db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user := new(types.User)
	for rows.Next() {
		user, err = scanRowIntoUser(rows)
		if err != nil {
			return nil, err
		}
	}

	if user.ID == uuid.Nil {
		return nil, types.ErrUserNotExist
	}

	return user, nil
}

func (s *Store) GetUserByUsername(username string) (*types.User, error) {
	statement := fmt.Sprintf("SELECT * FROM users WHERE username = '%s';", username)
	rows, err := s.db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user := new(types.User)
	for rows.Next() {
		user, err = scanRowIntoUser(rows)
		if err != nil {
			return nil, err
		}
	}

	if user.ID == uuid.Nil {
		return nil, types.ErrUserNotExist
	}

	return user, nil
}

func (s *Store) GetUserByID(id string) (*types.User, error) {
	statement := fmt.Sprintf("SELECT * FROM users WHERE id = '%s';", id)
	rows, err := s.db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user := new(types.User)
	for rows.Next() {
		user, err = scanRowIntoUser(rows)
		if err != nil {
			return nil, err
		}
	}

	if user.ID == uuid.Nil {
		return nil, types.ErrUserNotExist
	}

	return user, nil
}

func (s *Store) CreateUser(user types.User) error {
	createTime := user.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	statement := fmt.Sprintf(
		"INSERT INTO users ("+
			"id, username, firstname, lastname, "+
			"email, password_hash, "+
			"external_type, external_id, "+
			"create_time_utc, is_active"+
			") VALUES ('%s','%s','%s','%s','%s','%s','%s','%s','%s',%t);",
		user.ID, user.Username, user.Firstname, user.Lastname,
		user.Email, user.PasswordHashed,
		user.ExternalType, user.ExternalID,
		createTime, user.IsActive,
	)
	_, err := s.db.Exec(statement)
	if err != nil {
		return err
	}
	return nil
}

func scanRowIntoUser(rows *sql.Rows) (*types.User, error) {
	user := new(types.User)

	err := rows.Scan(
		&user.ID,
		&user.Username,
		&user.Firstname,
		&user.Lastname,
		&user.Email,
		&user.PasswordHashed,
		&user.ExternalType,
		&user.ExternalID,
		&user.CreateTime,
		&user.IsActive,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
