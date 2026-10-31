package session

import (
	"context"
	"database/sql"
	"errors"
	"uuid"

	"expense-tracker/agent/internal/models"
)

// ResolveThread separates untrusted UI identifiers from runtime-owned UUIDs.
// Retained anchors also prevent a deleted thread from being silently recreated.
func (s *Store) ResolveThread(ctx context.Context, app, owner, thread string, create bool) (models.Key, error) {
	key := models.Key{App: app, Owner: owner}
	if app == "" || owner == "" || thread == "" || len(owner) > 128 || len(thread) > 128 {
		return key, ErrPayload
	}
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `SELECT session_id FROM expense_agent.agui_ref WHERE app_name=$1 AND owner_id=$2 AND thread_id=$3 AND run_id IS NULL`, app, owner, thread).Scan(&key.Session)
		if err == nil {
			return lockActiveSession(ctx, tx, key)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return ErrStorage
		}
		if !create {
			return ErrMissing
		}
		key.Session = uuid.New().String()
		if _, err := tx.ExecContext(ctx, `INSERT INTO expense_agent.sessions(session_id,app_name,owner_id) VALUES($1,$2,$3)`, key.Session, app, owner); err != nil {
			return ErrStorage
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO expense_agent.agui_ref(id,app_name,owner_id,thread_id,session_id) VALUES($1,$2,$3,$4,$5)`, uuid.New().String(), app, owner, thread, key.Session)
		if err != nil {
			return ErrStorage
		}
		return nil
	})
	return key, err
}

func (s *Store) ResolveRun(ctx context.Context, key models.Key, thread, run string) (string, error) {
	var internal string
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockActiveSession(ctx, tx, key); err != nil {
			return err
		}
		err := tx.QueryRowContext(ctx, `SELECT runtime_run_id FROM expense_agent.agui_ref WHERE app_name=$1 AND owner_id=$2 AND thread_id=$3 AND run_id=$4 AND session_id=$5`, key.App, key.Owner, thread, run, key.Session).Scan(&internal)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMissing
		}
		if err != nil {
			return ErrStorage
		}
		return nil
	})
	return internal, err
}

func (s *Store) ListConversations(ctx context.Context, app, owner string) ([]models.Conversation, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT a.thread_id,GREATEST(s.updated_at,e.created_at) AS last_activity
FROM expense_agent.agui_ref a JOIN expense_agent.sessions s ON s.session_id=a.session_id
LEFT JOIN LATERAL (SELECT created_at FROM expense_agent.events WHERE session_id=s.session_id ORDER BY created_at DESC,id DESC LIMIT 1) e ON true
WHERE a.app_name=$1 AND a.owner_id=$2 AND a.run_id IS NULL AND s.deleted_at IS NULL
ORDER BY last_activity DESC,a.thread_id LIMIT 100`, app, owner)
	if err != nil {
		return nil, ErrStorage
	}
	defer rows.Close()
	items := []models.Conversation{}
	for rows.Next() {
		var item models.Conversation
		if rows.Scan(&item.ID, &item.UpdatedAt) != nil {
			return nil, ErrStorage
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrStorage
	}
	return items, nil
}
