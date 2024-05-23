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
	query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s';", email)
	rows, err := s.db.Query(query)
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
	query := fmt.Sprintf("SELECT * FROM users WHERE username = '%s';", username)
	rows, err := s.db.Query(query)
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
	query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s';", id)
	rows, err := s.db.Query(query)
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

func (s *Store) GetUsernameByID(userid string) (string, error) {
	query := fmt.Sprintf("SELECT username FROM users WHERE id='%s';", userid)
	rows, err := s.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var username string
	for rows.Next() {
		err := rows.Scan(&username)
		if err != nil {
			return "", err
		}
	}

	if username == "" {
		return "", types.ErrUserNotExist
	}

	return username, nil
}

func (s *Store) CreateUser(user types.User) error {
	createTime := user.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
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
	_, err := s.db.Exec(query)
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
