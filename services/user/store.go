package user

import (
	"database/sql"
	"expense-tracker/types"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetUserByEmail(email string) (*types.User, error) {
	query := "SELECT * FROM users WHERE email = $1;"
	rows, err := s.db.Query(query, email)
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
	query := "SELECT * FROM users WHERE id = $1;"
	rows, err := s.db.Query(query, id)
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
	query := "SELECT username FROM users WHERE id = $1;"
	rows, err := s.db.Query(query, userid)
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

func (s *Store) checkUserExist(query string, args ...interface{}) (bool, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return false, nil
	}
	defer rows.Close()

	exist := false
	for rows.Next() {
		err := rows.Scan(&exist)
		if err != nil {
			return false, err
		}
	}

	return exist, err
}
func (s *Store) CheckUserExistByEmail(email string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM users WHERE email = $1);"

	return s.checkUserExist(query, email)
}

func (s *Store) CheckUserExistByID(id string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM users WHERE id = $1);"

	return s.checkUserExist(query, id)
}

func (s *Store) CheckUserExistByUsername(username string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM users WHERE username = $1);"

	return s.checkUserExist(query, username)
}

func (s *Store) CheckEmailExist(email string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM users WHERE email = $1);"
	rows, err := s.db.Query(query, email)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	exist := false
	for rows.Next() {
		err := rows.Scan(&exist)
		if err != nil {
			return false, err
		}
	}

	return exist, nil
}

func (s *Store) CreateUser(user types.User) error {
	createTime := user.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := "INSERT INTO users (" +
		"id, username, firstname, lastname, nickname, " +
		"email, password_hash, " +
		"external_type, external_id, " +
		"create_time_utc, is_active, " +
		"role" +
		") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);"
	_, err := s.db.Exec(query,
		user.ID, user.Username, user.Firstname, user.Lastname, user.Nickname,
		user.Email, user.PasswordHashed,
		user.ExternalType, user.ExternalID,
		createTime, user.IsActive,
		user.Role)
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
		&user.Nickname,
		&user.Role,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
