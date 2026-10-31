// Package session defines durable execution contracts independently of transport
// and PostgreSQL. Raw diagnostic payloads are never returned as chat history.
package session

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"maps"
	"regexp"
	"strings"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
)

var (
	ErrMissing  = models.ErrMissing
	ErrConflict = models.ErrConflict
	ErrStorage  = models.ErrStorage
	ErrPayload  = models.ErrPayload
)

const MaxPayloadBytes = 128 << 10
const MaxHistoryBytes = 512 << 10

type bindingKey struct{}
type runBinding struct {
	Repo Repository
	Run  models.Run
}

func WithRun(ctx context.Context, repo Repository, run models.Run) context.Context {
	return context.WithValue(ctx, bindingKey{}, runBinding{repo, run})
}
func CurrentRun(ctx context.Context) (Repository, models.Run, bool) {
	b, ok := ctx.Value(bindingKey{}).(runBinding)
	return b.Repo, b.Run, ok
}
func RecordDiagnostic(ctx context.Context, kind, invocation string, value any) error {
	repo, run, ok := CurrentRun(ctx)
	if !ok {
		return nil
	}
	snap, err := captureDiagnostic(ctx, value)
	if err != nil {
		return err
	}
	if invocation == "" {
		if ref, ok := ctx.Value(callKey{}).(models.CallRef); ok {
			invocation = ref.InvocationID
		}
	}
	author := CurrentCall(ctx).Agent
	if agentContext, ok := ctx.(agent.InvocationContext); ok {
		author = agentContext.Agent().Name()
	}
	if fields, ok := value.(map[string]any); ok {
		if name, ok := fields["agent"].(string); ok && name != "" {
			author = name
		}
	}
	if event, ok := value.(*session.Event); ok {
		author = event.Author
	}
	_, _, err = repo.Append(ctx, run, models.Entry{Kind: kind, Author: author, InvocationID: invocation, Payload: snap})
	return err
}

// Keep correlation metadata in the envelope even when diagnostic content is
// omitted. Clone the map because model/tool writers may reuse it for later events.
func captureDiagnostic(ctx context.Context, value any) (models.Snapshot, error) {
	callID := ""
	if fields, ok := value.(map[string]any); ok {
		callID, _ = fields["callId"].(string)
		copy := maps.Clone(fields)
		delete(copy, "callId")
		value = copy
	}
	metadata, err := CaptureSnapshot(ctx, callID, true)
	if err != nil {
		return models.Snapshot{}, err
	}
	if json.Unmarshal(metadata.Data, &callID) != nil {
		return models.Snapshot{}, ErrPayload
	}
	snap, err := CaptureSnapshot(ctx, value, false)
	if err != nil {
		return models.Snapshot{}, err
	}
	snap.CallID = callID
	return snap, nil
}

type callKey struct{}

func WithCall(ctx context.Context, ref models.CallRef) context.Context {
	return context.WithValue(ctx, callKey{}, ref)
}
func CurrentCall(ctx context.Context) models.CallRef {
	ref, _ := ctx.Value(callKey{}).(models.CallRef)
	return ref
}

var jwtPattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
var bearerPattern = regexp.MustCompile(`(?i)Bearer\s+[^\s"\\]+`)
var assignmentPattern = regexp.MustCompile(`(?i)(api[_-]?key|password|secret|access[_-]?token|refresh[_-]?token)\s*[:=]\s*[^\s,;"\\]+`)
var credentialURLPattern = regexp.MustCompile(`(?i)(postgres(?:ql)?|https?)://[^\s:/]+:[^\s@]+@`)

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ":", "").Replace(key))
	return strings.Contains(normalized, "password") || strings.Contains(normalized, "apikey") || strings.Contains(normalized, "authorization") || strings.Contains(normalized, "cookie") || strings.Contains(normalized, "secret") || normalized == "accesstoken" || normalized == "refreshtoken" || normalized == "credentials" || normalized == "httpoptions"
}
func redactSecrets(ctx context.Context, value any) any {
	switch v := value.(type) {
	case map[string]any:
		for k, item := range v {
			if isSensitiveKey(k) {
				v[k] = "[REDACTED]"
			} else {
				v[k] = redactSecrets(ctx, item)
			}
		}
		return v
	case []any:
		for i := range v {
			v[i] = redactSecrets(ctx, v[i])
		}
		return v
	case string:
		secrets := credentials.Secrets(ctx)
		for _, s := range secrets {
			if s != "" {
				v = strings.ReplaceAll(v, s, "[REDACTED]")
			}
		}
		v = jwtPattern.ReplaceAllString(v, "[REDACTED]")
		v = bearerPattern.ReplaceAllString(v, "[REDACTED]")
		v = credentialURLPattern.ReplaceAllString(v, "[REDACTED-URL]@")
		return assignmentPattern.ReplaceAllString(v, "[REDACTED]")
	default:
		return value
	}
}

// CaptureSnapshot omits oversized diagnostic content explicitly. Required execution
// history instead fails closed, so signatures/state are never silently clipped.
func CaptureSnapshot(ctx context.Context, value any, required bool) (models.Snapshot, error) {
	original, err := json.Marshal(value)
	if err != nil {
		return models.Snapshot{}, ErrPayload
	}
	var decoded any
	if json.Unmarshal(original, &decoded) != nil {
		return models.Snapshot{}, ErrPayload
	}
	data, err := json.Marshal(redactSecrets(ctx, decoded))
	if err != nil {
		return models.Snapshot{}, ErrPayload
	}
	snap := models.Snapshot{Data: data, Complete: true, OriginalBytes: len(original), RetainedBytes: len(data)}
	if len(data) > MaxPayloadBytes {
		if required {
			return models.Snapshot{}, ErrPayload
		}
		snap.Data = jsontext.Value(`null`)
		snap.Complete = false
		snap.RetainedBytes = 0
		snap.Reason = "payload_limit"
	}
	return snap, nil
}
