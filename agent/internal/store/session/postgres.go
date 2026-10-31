// Package session owns the scoped PostgreSQL Agent state boundary.
package session

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"expense-tracker/agent/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const operationTimeout = 3 * time.Second

type Store struct {
	db          *sql.DB
	outputGates sync.Map
}

func New(db *sql.DB) *Store { return &Store{db: db} }
func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, ErrStorage
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(15 * time.Minute)
	db.SetConnMaxIdleTime(time.Minute)
	store := New(db)
	if err := store.VerifyRole(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) VerifyRole(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	var safe bool
	err := s.db.QueryRowContext(ctx, `SELECT NOT (rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls OR rolinherit)
 AND has_schema_privilege(current_user,'expense_agent','USAGE')
 AND NOT has_schema_privilege(current_user,'expense_agent','CREATE')
 AND NOT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname='public' AND
 (has_table_privilege(current_user,format('%I.%I',schemaname,tablename),'SELECT') OR
 has_table_privilege(current_user,format('%I.%I',schemaname,tablename),'INSERT,UPDATE,DELETE')))
 AND has_table_privilege(current_user,'expense_agent.sessions','SELECT,INSERT,UPDATE')
 FROM pg_roles WHERE rolname=current_user`).Scan(&safe)
	if err != nil || !safe {
		return errors.New("agent database role is not isolated")
	}
	return nil
}
func (s *Store) transaction(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return ErrStorage
	}
	defer tx.Rollback()
	if err = fn(ctx, tx); err != nil {
		return err
	}
	if tx.Commit() != nil {
		return ErrStorage
	}
	return nil
}
func lockActiveSession(ctx context.Context, tx *sql.Tx, key models.Key) error {
	var deleted sql.NullTime
	err := tx.QueryRowContext(ctx, `SELECT deleted_at FROM expense_agent.sessions WHERE app_name=$1 AND owner_id=$2 AND session_id=$3 FOR UPDATE`, key.App, key.Owner, key.Session).Scan(&deleted)
	if errors.Is(err, sql.ErrNoRows) || deleted.Valid {
		return ErrMissing
	}
	if err != nil {
		return ErrStorage
	}
	return nil
}
