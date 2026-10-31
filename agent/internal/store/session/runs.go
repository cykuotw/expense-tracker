package session

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"time"
	"uuid"

	"expense-tracker/agent/internal/models"
)

// requireRunMapping validates the durable identity mapping, not execution ownership.
// MVP admission and resend protection live in the single runtime process.
func requireRunMapping(ctx context.Context, tx *sql.Tx, run models.Run) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM expense_agent.agui_ref WHERE runtime_run_id=$1 AND app_name=$2 AND owner_id=$3 AND session_id=$4)`, run.ID, run.App, run.Owner, run.Session).Scan(&exists); err != nil {
		return ErrStorage
	}
	if !exists {
		return ErrMissing
	}
	return nil
}

func (s *Store) Begin(ctx context.Context, key models.Key, message string, input any) (models.Run, error) {
	return s.begin(ctx, key, models.UIRef{ThreadID: key.Session, RunID: uuid.New().String()}, message, input, true)
}

func (s *Store) BeginUI(ctx context.Context, key models.Key, ref models.UIRef, message string, input any) (models.Run, error) {
	return s.begin(ctx, key, ref, message, input, false)
}

func (s *Store) begin(ctx context.Context, key models.Key, ref models.UIRef, message string, input any, native bool) (models.Run, error) {
	if !validKey(key) || message == "" || ref.ThreadID == "" || ref.RunID == "" || len(ref.ThreadID) > 128 || len(ref.RunID) > 128 {
		return models.Run{}, ErrPayload
	}
	snap, err := CaptureSnapshot(ctx, input, true)
	if err != nil {
		return models.Run{}, err
	}
	var text struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(snap.Data, &text) != nil {
		return models.Run{}, ErrPayload
	}
	run := models.Run{Key: key, UI: ref, ID: uuid.New().String(), MessageID: uuid.New().String(), ClientMessageID: message, Text: text.Text, Status: "running", StartedAt: time.Now().UTC()}
	err = s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if native {
			if _, err := tx.ExecContext(ctx, `INSERT INTO expense_agent.sessions(app_name,owner_id,session_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, key.App, key.Owner, key.Session); err != nil {
				return ErrStorage
			}
		}
		if err := lockActiveSession(ctx, tx, key); err != nil {
			return err
		}
		if native {
			if _, err := tx.ExecContext(ctx, `INSERT INTO expense_agent.agui_ref(id,app_name,owner_id,thread_id,session_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, uuid.New().String(), key.App, key.Owner, ref.ThreadID, key.Session); err != nil {
				return ErrStorage
			}
		}
		var mapped bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM expense_agent.agui_ref WHERE app_name=$1 AND owner_id=$2 AND thread_id=$3 AND session_id=$4 AND run_id IS NULL)`, key.App, key.Owner, ref.ThreadID, key.Session).Scan(&mapped); err != nil {
			return ErrStorage
		}
		if !mapped {
			return ErrMissing
		}
		var existing string
		err := tx.QueryRowContext(ctx, `SELECT runtime_run_id FROM expense_agent.agui_ref WHERE app_name=$1 AND owner_id=$2 AND thread_id=$3 AND run_id=$4 AND session_id=$5`, key.App, key.Owner, ref.ThreadID, ref.RunID, key.Session).Scan(&existing)
		if err == nil {
			run.ID = existing
		} else if errors.Is(err, sql.ErrNoRows) {
			if _, err := tx.ExecContext(ctx, `INSERT INTO expense_agent.agui_ref(id,app_name,owner_id,thread_id,run_id,session_id,runtime_run_id) VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.New().String(), key.App, key.Owner, ref.ThreadID, ref.RunID, key.Session, run.ID); err != nil {
				return ErrStorage
			}
		} else {
			return ErrStorage
		}
		// A reused UI ID keeps its lookup mapping. Whether a resend may execute
		// is decided by the runtime's in-memory guard, never a DB admission rule.
		data, err := CaptureSnapshot(ctx, run, true)
		if err != nil {
			return err
		}
		_, run.StartedAt, err = appendTx(ctx, tx, run, models.Entry{Kind: "run.started", Author: "user", Payload: data})
		return err
	})
	return run, err
}

func appendTx(ctx context.Context, tx *sql.Tx, run models.Run, entry models.Entry) (string, time.Time, error) {
	// Canonical ADK events belong to the session writer. Execution
	// snapshots must never be read as ADK history.
	if entry.Kind == "adk.event" {
		return "", time.Time{}, ErrPayload
	}
	payload, err := json.Marshal(entry.Payload)
	if err != nil {
		return "", time.Time{}, ErrPayload
	}
	return appendEventTx(ctx, tx, run.Key, storedEvent{RunID: run.ID, EventType: entry.Kind, Author: entry.Author,
		InvocationID: entry.InvocationID, Data: payload})
}

func (s *Store) Append(ctx context.Context, run models.Run, entry models.Entry) (string, time.Time, error) {
	var id string
	var createdAt time.Time
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockActiveSession(ctx, tx, run.Key); err != nil {
			return err
		}
		if err := requireRunMapping(ctx, tx, run); err != nil {
			return err
		}
		var err error
		id, createdAt, err = appendTx(ctx, tx, run, entry)
		return err
	})
	return id, createdAt, err
}
func finishTx(ctx context.Context, tx *sql.Tx, run models.Run, status, code string) error {
	data, err := CaptureSnapshot(ctx, struct {
		Status    string `json:"status"`
		ErrorCode string `json:"errorCode,omitempty"`
	}{status, code}, true)
	if err != nil {
		return err
	}
	_, _, err = appendTx(ctx, tx, run, models.Entry{Kind: "run.finished", Payload: data})
	return err
}

func (s *Store) Finish(ctx context.Context, run models.Run, status, code string) error {
	return s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockActiveSession(ctx, tx, run.Key); err != nil {
			return err
		}
		if err := requireRunMapping(ctx, tx, run); err != nil {
			return err
		}
		return finishTx(ctx, tx, run, status, code)
	})
}

// withSessionOutputLock coordinates output and deletion within this Store's single runtime.
// It is not a distributed lock or a cross-restart execution guarantee.
func (s *Store) withSessionOutputLock(ctx context.Context, key models.Key, fn func(context.Context, *sql.Conn) error) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	value, _ := s.outputGates.LoadOrStore(key.Session, make(chan struct{}, 1))
	gate := value.(chan struct{})
	select {
	case gate <- struct{}{}:
		defer func() { <-gate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return ErrStorage
	}
	defer conn.Close()
	return fn(ctx, conn)
}

func (s *Store) PersistAndEmitOutput(ctx context.Context, run models.Run, output models.Output, emit func(models.Output) error) error {
	return s.withSessionOutputLock(ctx, run.Key, func(ctx context.Context, conn *sql.Conn) error {
		tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			return ErrStorage
		}
		defer tx.Rollback()
		if err := lockActiveSession(ctx, tx, run.Key); err != nil {
			return err
		}
		if err := requireRunMapping(ctx, tx, run); err != nil {
			return err
		}
		snap, err := CaptureSnapshot(ctx, output, true)
		if err != nil {
			return err
		}
		output.EventID, output.CreatedAt, err = appendTx(ctx, tx, run, models.Entry{Kind: "output", Author: output.Skill, InvocationID: output.InvocationID, Payload: snap})
		if err != nil {
			return err
		}
		if output.Kind == "finished" || output.Kind == "error" {
			status := "completed"
			if output.Kind == "error" {
				status = "failed"
				if output.ErrorCode == "interrupted" {
					status = "interrupted"
				}
			}
			if err := finishTx(ctx, tx, run, status, output.ErrorCode); err != nil {
				return ErrStorage
			}
		}
		if tx.Commit() != nil {
			return ErrStorage
		}
		return emit(output)
	})
}
func (s *Store) SoftDelete(ctx context.Context, key models.Key) error {
	return s.withSessionOutputLock(ctx, key, func(ctx context.Context, conn *sql.Conn) error {
		tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			return ErrStorage
		}
		defer tx.Rollback()
		if err := lockActiveSession(ctx, tx, key); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE expense_agent.sessions SET deleted_at=NOW(),updated_at=NOW() WHERE app_name=$1 AND owner_id=$2 AND session_id=$3`, key.App, key.Owner, key.Session); err != nil {
			return ErrStorage
		}

		if tx.Commit() != nil {
			return ErrStorage
		}
		return nil
	})
}
func (s *Store) ListSessions(ctx context.Context, app, owner string) ([]models.Conversation, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT s.session_id,GREATEST(s.updated_at,e.created_at) AS last_activity
FROM expense_agent.sessions s LEFT JOIN LATERAL
(SELECT created_at FROM expense_agent.events WHERE session_id=s.session_id ORDER BY created_at DESC,id DESC LIMIT 1) e ON true
WHERE s.app_name=$1 AND s.owner_id=$2 AND s.deleted_at IS NULL ORDER BY last_activity DESC,s.session_id LIMIT 100`, app, owner)
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
func (s *Store) ListRuns(ctx context.Context, key models.Key) ([]models.Run, error) {
	items := []models.Run{}
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockActiveSession(ctx, tx, key); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT started.data,started.created_at,a.thread_id,a.run_id,finished.data
FROM (SELECT DISTINCT ON (run_id) run_id,data,created_at FROM expense_agent.events
 WHERE session_id=$1 AND event_type='run.started'
 ORDER BY run_id,created_at DESC,id DESC) started
JOIN expense_agent.agui_ref a ON a.runtime_run_id=started.run_id
LEFT JOIN LATERAL (SELECT data FROM expense_agent.events
 WHERE run_id=started.run_id AND event_type='run.finished' AND created_at>started.created_at
 ORDER BY created_at DESC,id DESC LIMIT 1) finished ON true
ORDER BY started.created_at DESC LIMIT 100`, key.Session)
		if err != nil {
			return ErrStorage
		}
		defer rows.Close()
		for rows.Next() {
			var startData, endData []byte
			var startedAt time.Time
			var ref models.UIRef
			if rows.Scan(&startData, &startedAt, &ref.ThreadID, &ref.RunID, &endData) != nil {
				return ErrStorage
			}
			var snap models.Snapshot
			var run models.Run
			if json.Unmarshal(startData, &snap) != nil || !snap.Complete || json.Unmarshal(snap.Data, &run) != nil {
				return ErrStorage
			}
			run.Key, run.UI, run.StartedAt = key, ref, startedAt.UTC()
			if endData != nil {
				var terminal struct {
					Status    string `json:"status"`
					ErrorCode string `json:"errorCode"`
				}
				if json.Unmarshal(endData, &snap) != nil || !snap.Complete || json.Unmarshal(snap.Data, &terminal) != nil {
					return ErrStorage
				}
				run.Status, run.ErrorCode = terminal.Status, terminal.ErrorCode
			}
			items = append(items, run)
		}
		if rows.Err() != nil {
			return ErrStorage
		}
		return nil
	})
	return items, err
}
func (s *Store) ReadRunOutput(ctx context.Context, key models.Key, runID, after string) ([]models.Output, error) {
	items := []models.Output{}
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockActiveSession(ctx, tx, key); err != nil {
			return err
		}
		var exists bool
		if tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM expense_agent.agui_ref WHERE runtime_run_id=$1 AND app_name=$2 AND owner_id=$3 AND session_id=$4)`, runID, key.App, key.Owner, key.Session).Scan(&exists) != nil {
			return ErrStorage
		}
		if !exists {
			return ErrMissing
		}
		var afterTime time.Time
		var afterID uuid.UUID
		if after != "" {
			var err error
			afterID, err = uuid.Parse(after)
			if err != nil {
				return ErrPayload
			}
			err = tx.QueryRowContext(ctx, `SELECT created_at FROM expense_agent.events WHERE id=$1 AND run_id=$2 AND event_type='output'`, afterID.String(), runID).Scan(&afterTime)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrMissing
			}
			if err != nil {
				return ErrStorage
			}
		}
		rows, err := tx.QueryContext(ctx, `SELECT id,created_at,data FROM expense_agent.events WHERE run_id=$1 AND event_type='output' AND ($2 OR (created_at,id)>($3,$4::uuid)) ORDER BY created_at,id LIMIT 256`, runID, after == "", afterTime, afterID.String())
		if err != nil {
			return ErrStorage
		}
		defer rows.Close()
		size := 0
		for rows.Next() {
			var data []byte
			var id string
			var createdAt time.Time
			if rows.Scan(&id, &createdAt, &data) != nil {
				return ErrStorage
			}
			size += len(data)
			if size > MaxHistoryBytes {
				return ErrPayload
			}
			var snap models.Snapshot
			var output models.Output
			if json.Unmarshal(data, &snap) != nil || json.Unmarshal(snap.Data, &output) != nil {
				return ErrStorage
			}
			output.EventID = id
			output.CreatedAt = createdAt.UTC()
			items = append(items, output)
		}
		if rows.Err() != nil {
			return ErrStorage
		}
		return nil
	})
	return items, err
}
