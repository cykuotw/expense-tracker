package user

import (
	"database/sql"
	"errors"
	"strings"

	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
)

type Store struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...any) error
}

const userProjection = `
	id, username, firstname, lastname, email, password_hash,
	has_local_password, external_type, external_id, create_time_utc, is_active, nickname, role
`

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetUserByEmail(email string) (*types.User, error) {
	query := "SELECT " + userProjection + " FROM users WHERE LOWER(BTRIM(email)) = $1;"
	user, err := scanRowIntoUser(s.db.QueryRow(query, auth.NormalizeEmail(email)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, types.ErrUserNotExist
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Store) GetUserByExternalIdentity(externalType string, externalID string) (*types.User, error) {
	query := "SELECT " + userProjection + " FROM users WHERE external_type = $1 AND external_id = $2;"
	user, err := scanRowIntoUser(s.db.QueryRow(query, externalType, externalID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, types.ErrUserNotExist
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Store) GetUserByID(id string) (*types.User, error) {
	query := "SELECT " + userProjection + " FROM users WHERE id = $1;"
	user, err := scanRowIntoUser(s.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, types.ErrUserNotExist
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Store) GetUsernameByID(userid string) (string, error) {
	query := `SELECT COALESCE(
		NULLIF(BTRIM(nickname), ''),
		NULLIF(BTRIM(CONCAT_WS(' ', firstname, lastname)), ''),
		username
	) FROM users WHERE id = $1;`
	var username string
	if err := s.db.QueryRow(query, userid).Scan(&username); errors.Is(err, sql.ErrNoRows) {
		return "", types.ErrUserNotExist
	} else if err != nil {
		return "", err
	}

	return username, nil
}

func (s *Store) GetUsernamesByIDs(userIDs []string) (map[string]string, error) {
	if len(userIDs) == 0 {
		return map[string]string{}, nil
	}

	rows, err := s.db.Query(
		`SELECT id,
			COALESCE(
				NULLIF(BTRIM(nickname), ''),
				NULLIF(BTRIM(CONCAT_WS(' ', firstname, lastname)), ''),
				username
			)
		FROM users
		WHERE id = ANY($1::uuid[]);`,
		userIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	usernames := make(map[string]string, len(userIDs))
	for rows.Next() {
		var id, username string
		if err := rows.Scan(&id, &username); err != nil {
			return nil, err
		}
		usernames[id] = username
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return usernames, nil
}

func (s *Store) checkUserExist(query string, args ...any) (bool, error) {
	exist := false
	if err := s.db.QueryRow(query, args...).Scan(&exist); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return exist, nil
}
func (s *Store) CheckUserExistByEmail(email string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM users WHERE LOWER(BTRIM(email)) = $1);"

	return s.checkUserExist(query, auth.NormalizeEmail(email))
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
	query := "SELECT EXISTS (SELECT 1 FROM users WHERE LOWER(BTRIM(email)) = $1);"
	return s.checkUserExist(query, auth.NormalizeEmail(email))
}

func (s *Store) CreateUser(user types.User) error {
	createTime := user.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := "INSERT INTO users (" +
		"id, username, firstname, lastname, nickname, " +
		"email, password_hash, has_local_password, " +
		"external_type, external_id, " +
		"create_time_utc, is_active, " +
		"role" +
		") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);"
	_, err := s.db.Exec(query,
		user.ID, user.Username, user.Firstname, user.Lastname, user.Nickname,
		auth.NormalizeEmail(user.Email), user.PasswordHashed, user.HasLocalPassword,
		nullableString(user.ExternalType), nullableString(user.ExternalID),
		createTime, user.IsActive,
		user.Role)
	if err != nil {
		return err
	}
	return nil
}

func scanRowIntoUser(row rowScanner) (*types.User, error) {
	user := new(types.User)
	var externalType sql.NullString
	var externalID sql.NullString

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Firstname,
		&user.Lastname,
		&user.Email,
		&user.PasswordHashed,
		&user.HasLocalPassword,
		&externalType,
		&externalID,
		&user.CreateTime,
		&user.IsActive,
		&user.Nickname,
		&user.Role,
	)
	if err != nil {
		return nil, err
	}
	user.ExternalType = nullStringToString(externalType)
	user.ExternalID = nullStringToString(externalID)
	return user, nil
}

func nullableString(value string) any {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}
	return normalized
}

func nullStringToString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
