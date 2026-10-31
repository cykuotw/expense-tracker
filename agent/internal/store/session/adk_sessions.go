package session

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"iter"
	"maps"
	"strings"
	"sync"
	"time"
	"uuid"

	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/session"
)

type state struct {
	mu     sync.RWMutex
	values map[string]any
}

func (s *state) Get(k string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[k]
	if !ok {
		return nil, session.ErrStateKeyNotExist
	}
	return v, nil
}
func (s *state) Set(k string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[k] = v
	return nil
}
func (s *state) All() iter.Seq2[string, any] {
	s.mu.RLock()
	copy := maps.Clone(s.values)
	s.mu.RUnlock()
	return maps.All(copy)
}

type eventList []*session.Event

func (e eventList) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, v := range e {
			if !yield(v) {
				return
			}
		}
	}
}
func (e eventList) Len() int                { return len(e) }
func (e eventList) At(i int) *session.Event { return e[i] }

type storedSession struct {
	key     models.Key
	state   *state
	events  eventList
	updated time.Time
}

func (s *storedSession) ID() string                { return s.key.Session }
func (s *storedSession) AppName() string           { return s.key.App }
func (s *storedSession) UserID() string            { return s.key.Owner }
func (s *storedSession) State() session.State      { return s.state }
func (s *storedSession) Events() session.Events    { return s.events }
func (s *storedSession) LastUpdateTime() time.Time { return s.updated }
func keyOf(s session.Session) models.Key {
	return models.Key{App: s.AppName(), Owner: s.UserID(), Session: s.ID()}
}
func validKey(k models.Key) bool {
	_, err := uuid.Parse(k.Session)
	return err == nil && k.App != "" && k.Owner != "" && len(k.Owner) <= 128
}

func (s *Store) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if req == nil {
		return nil, ErrPayload
	}
	key := models.Key{App: req.AppName, Owner: req.UserID, Session: req.SessionID}
	if key.Session == "" {
		key.Session = uuid.New().String()
	}
	if !validKey(key) {
		return nil, ErrPayload
	}
	values := map[string]any{}
	for k, v := range req.State {
		if strings.HasPrefix(k, "temp:") {
			continue
		}
		if strings.HasPrefix(k, "app:") || strings.HasPrefix(k, "user:") {
			return nil, ErrPayload
		}
		values[k] = v
	}
	snap, err := CaptureSnapshot(ctx, values, true)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO expense_agent.sessions(app_name,owner_id,session_id,state) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, key.App, key.Owner, key.Session, []byte(snap.Data))
	if err != nil {
		return nil, ErrStorage
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return nil, ErrConflict
	}
	loaded, err := s.Get(ctx, &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	if err != nil {
		return nil, err
	}
	return &session.CreateResponse{Session: loaded.Session}, nil
}
func (s *Store) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	if req == nil {
		return nil, ErrPayload
	}
	key := models.Key{App: req.AppName, Owner: req.UserID, Session: req.SessionID}
	if !validKey(key) {
		return nil, ErrPayload
	}
	var loaded *storedSession
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockActiveSession(ctx, tx, key); err != nil {
			if errors.Is(err, ErrMissing) {
				return session.ErrNotFound
			}
			return err
		}
		var data []byte
		var updated time.Time
		if tx.QueryRowContext(ctx, `SELECT state,GREATEST(updated_at, (SELECT created_at FROM expense_agent.events WHERE session_id=$3 ORDER BY created_at DESC,id DESC LIMIT 1)) FROM expense_agent.sessions WHERE app_name=$1 AND owner_id=$2 AND session_id=$3`, key.App, key.Owner, key.Session).Scan(&data, &updated) != nil {
			return ErrStorage
		}
		values := map[string]any{}
		if json.Unmarshal(data, &values) != nil {
			return ErrStorage
		}
		rows, err := tx.QueryContext(ctx, `SELECT data FROM (SELECT created_at,id,data FROM expense_agent.events WHERE session_id=$1 AND event_type='adk.event' ORDER BY created_at DESC,id DESC LIMIT $2) AS recent ORDER BY created_at,id`, key.Session, 256)
		if err != nil {
			return ErrStorage
		}
		defer rows.Close()
		events := eventList{}
		size := len(data)
		for rows.Next() {
			var payload []byte
			if rows.Scan(&payload) != nil {
				return ErrStorage
			}
			size += len(payload)
			if size > MaxHistoryBytes {
				return ErrPayload
			}
			event := new(session.Event)
			if json.Unmarshal(payload, event) != nil {
				return ErrStorage
			}
			if !req.After.IsZero() && event.Timestamp.Before(req.After) {
				continue
			}
			events = append(events, event)
		}
		if rows.Err() != nil {
			return ErrStorage
		}
		if req.NumRecentEvents > 0 && len(events) > req.NumRecentEvents {
			events = events[len(events)-req.NumRecentEvents:]
		}
		loaded = &storedSession{key: key, state: &state{values: values}, events: events, updated: updated.UTC()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &session.GetResponse{Session: loaded}, nil
}
func (s *Store) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	if req == nil {
		return nil, ErrPayload
	}
	items, err := s.ListSessions(ctx, req.AppName, req.UserID)
	if err != nil {
		return nil, err
	}
	result := &session.ListResponse{}
	for _, item := range items {
		loaded, err := s.Get(ctx, &session.GetRequest{AppName: req.AppName, UserID: req.UserID, SessionID: item.ID})
		if err != nil {
			return nil, err
		}
		result.Sessions = append(result.Sessions, loaded.Session)
	}
	return result, nil
}
func (s *Store) Delete(ctx context.Context, req *session.DeleteRequest) error {
	if req == nil {
		return ErrPayload
	}
	err := s.SoftDelete(ctx, models.Key{App: req.AppName, Owner: req.UserID, Session: req.SessionID})
	if errors.Is(err, ErrMissing) {
		return nil
	}
	return err
}
func (s *Store) AppendEvent(ctx context.Context, current session.Session, event *session.Event) error {
	if current == nil || event == nil {
		return ErrPayload
	}
	if event.Partial {
		return nil
	}
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	copy := *event
	copy.Actions = event.Actions
	copy.Actions.StateDelta = map[string]any{}
	for k, v := range event.Actions.StateDelta {
		if strings.HasPrefix(k, "temp:") {
			continue
		}
		if strings.HasPrefix(k, "app:") || strings.HasPrefix(k, "user:") {
			return ErrPayload
		}
		copy.Actions.StateDelta[k] = v
	}
	snap, err := CaptureSnapshot(ctx, &copy, true)
	if err != nil {
		return err
	}
	var sanitized session.Event
	if json.Unmarshal(snap.Data, &sanitized) != nil {
		return ErrPayload
	}
	key := keyOf(current)
	values := map[string]any{}
	err = s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockActiveSession(ctx, tx, key); err != nil {
			return err
		}
		runID := ""
		if _, run, bound := CurrentRun(ctx); bound {
			if run.Key != key {
				return ErrConflict
			}
			if err := requireRunMapping(ctx, tx, run); err != nil {
				return err
			}
			runID = run.ID
		}
		var data []byte
		if tx.QueryRowContext(ctx, `SELECT state FROM expense_agent.sessions WHERE app_name=$1 AND owner_id=$2 AND session_id=$3`, key.App, key.Owner, key.Session).Scan(&data) != nil {
			return ErrStorage
		}
		if json.Unmarshal(data, &values) != nil {
			return ErrStorage
		}
		for k, v := range sanitized.Actions.StateDelta {
			values[k] = v
		}
		stateData, err := CaptureSnapshot(ctx, values, true)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE expense_agent.sessions SET state=$4,updated_at=NOW() WHERE app_name=$1 AND owner_id=$2 AND session_id=$3`, key.App, key.Owner, key.Session, []byte(stateData.Data))
		if err != nil {
			return ErrStorage
		}
		_, _, err = appendEventTx(ctx, tx, key, storedEvent{RunID: runID, EventType: "adk.event", Author: sanitized.Author, InvocationID: sanitized.InvocationID, Data: snap.Data})
		if err != nil {
			return ErrStorage
		}
		return nil
	})
	if err != nil {
		return err
	}
	for k, v := range values {
		current.State().Set(k, v)
	}
	if loaded, ok := current.(*storedSession); ok {
		loaded.events = append(loaded.events, &sanitized)
		loaded.updated = time.Now().UTC()
	}
	return nil
}
