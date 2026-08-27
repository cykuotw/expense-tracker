package user

import (
	"context"
	"database/sql"
	"expense-tracker/backend/types"
	"fmt"
)

const adminStateLockID int64 = 726814993

func validateProtectedAdminMutation(protected bool) error {
	if protected {
		return types.ErrProtectedAdmin
	}
	return nil
}

func validateStatusChange(actorID, targetID, role string, currentlyActive, nextActive bool, activeAdmins int, protected bool) error {
	if err := validateProtectedAdminMutation(protected); err != nil {
		return err
	}
	if nextActive {
		return nil
	}
	if actorID == targetID {
		return types.ErrCannotDeactivateSelf
	}
	if currentlyActive && role == "admin" && activeAdmins <= 1 {
		return types.ErrLastActiveAdmin
	}
	return nil
}

func validateRoleChange(actorID, targetID, currentRole, nextRole string, active bool, activeAdmins int, protected bool) error {
	if nextRole != "admin" && nextRole != "user" {
		return types.ErrInvalidUserRole
	}
	if err := validateProtectedAdminMutation(protected); err != nil {
		return err
	}
	if actorID == targetID {
		return types.ErrCannotChangeOwnRole
	}
	if currentRole == "admin" && nextRole == "user" && active && activeAdmins <= 1 {
		return types.ErrLastActiveAdmin
	}
	return nil
}

func (s *Store) GetAdminUsers() ([]types.AdminUserResponse, error) {
	rows, err := s.db.Query(`
		SELECT id, firstname, lastname, email, nickname, role, is_active, is_protected_admin, create_time_utc
		FROM users
		ORDER BY create_time_utc DESC, email ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []types.AdminUserResponse{}
	for rows.Next() {
		var user types.AdminUserResponse
		if err := rows.Scan(
			&user.ID, &user.Firstname, &user.Lastname, &user.Email,
			&user.Nickname, &user.Role, &user.IsActive, &user.IsProtectedAdmin, &user.CreateTime,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Store) SetUserActive(actorID string, targetID string, active bool) error {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1);", adminStateLockID); err != nil {
		return err
	}

	var role string
	var currentlyActive bool
	var protected bool
	if err := tx.QueryRow(
		"SELECT role, is_active, is_protected_admin FROM users WHERE id = $1 FOR UPDATE;",
		targetID,
	).Scan(&role, &currentlyActive, &protected); err != nil {
		if err == sql.ErrNoRows {
			return types.ErrUserNotExist
		}
		return err
	}
	if err := validateProtectedAdminMutation(protected); err != nil {
		return err
	}

	activeAdmins := 0
	if !active && currentlyActive && role == "admin" {
		if err := tx.QueryRow(
			"SELECT COUNT(*) FROM users WHERE role = 'admin' AND is_active = TRUE;",
		).Scan(&activeAdmins); err != nil {
			return err
		}
	}
	if err := validateStatusChange(actorID, targetID, role, currentlyActive, active, activeAdmins, false); err != nil {
		return err
	}

	result, err := tx.Exec("UPDATE users SET is_active = $2 WHERE id = $1;", targetID, active)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected updated user count: %d", affected)
	}

	if !active {
		if _, err := tx.Exec(
			"UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL;",
			targetID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) SetUserRole(actorID string, targetID string, role string) error {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1);", adminStateLockID); err != nil {
		return err
	}

	var currentRole string
	var active bool
	var protected bool
	if err := tx.QueryRow(
		"SELECT role, is_active, is_protected_admin FROM users WHERE id = $1 FOR UPDATE;",
		targetID,
	).Scan(&currentRole, &active, &protected); err != nil {
		if err == sql.ErrNoRows {
			return types.ErrUserNotExist
		}
		return err
	}
	if err := validateProtectedAdminMutation(protected); err != nil {
		return err
	}

	activeAdmins := 0
	if currentRole == "admin" && role == "user" && active {
		if err := tx.QueryRow(
			"SELECT COUNT(*) FROM users WHERE role = 'admin' AND is_active = TRUE;",
		).Scan(&activeAdmins); err != nil {
			return err
		}
	}
	if err := validateRoleChange(actorID, targetID, currentRole, role, active, activeAdmins, false); err != nil {
		return err
	}

	result, err := tx.Exec("UPDATE users SET role = $2 WHERE id = $1;", targetID, role)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected updated user count: %d", affected)
	}
	return tx.Commit()
}
