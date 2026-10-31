package session

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"time"
	"uuid"

	"expense-tracker/agent/internal/models"
)

type storedEvent struct {
	RunID, EventType, Author, InvocationID string
	Data                                   []byte
}

// appendEventTx orders events by database creation time without updating a counter.
// Callers hold the session row lock through lockActiveSession; state changes and event
// insertion share the same transaction and roll back together.
func appendEventTx(ctx context.Context, tx *sql.Tx, key models.Key, event storedEvent) (string, time.Time, error) {
	if event.Author == "" {
		event.Author = "system"
	}
	author, err := CaptureSnapshot(ctx, event.Author, true)
	if err != nil || json.Unmarshal(author.Data, &event.Author) != nil || len(event.Author) > 256 {
		return "", time.Time{}, ErrPayload
	}
	id := uuid.New().String()
	var createdAt time.Time
	// lockActiveSession acquired the session lock in an earlier statement. The
	// latest-event lookup observes the preceding writer's committed timestamp.
	// Advance by one microsecond only when the clock ties or moves backwards;
	// source/ADK timestamps remain unchanged inside the event data.
	err = tx.QueryRowContext(ctx, `INSERT INTO expense_agent.events(id,session_id,run_id,event_type,author,invocation_id,data,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,
		GREATEST(clock_timestamp(), (SELECT created_at + INTERVAL '1 microsecond' FROM expense_agent.events WHERE session_id=$2 ORDER BY created_at DESC,id DESC LIMIT 1)))
		RETURNING created_at`,
		id, key.Session, sql.NullString{String: event.RunID, Valid: event.RunID != ""},
		event.EventType, event.Author, event.InvocationID, event.Data).Scan(&createdAt)
	if err != nil {
		return "", time.Time{}, ErrStorage
	}
	return id, createdAt.UTC(), nil
}
