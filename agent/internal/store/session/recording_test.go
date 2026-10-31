package session

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/agent/internal/models"

	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestExecutionRecordingRejectsHistoryType(t *testing.T) {
	for _, eventType := range []string{"adk.event"} {
		t.Run(eventType, func(t *testing.T) {
			// Reject before touching storage: a diagnostic snapshot is neither a
			// native ADK event, even if its type matches.
			_, _, err := appendTx(t.Context(), nil, models.Run{}, models.Entry{Kind: eventType})
			require.ErrorIs(t, err, ErrPayload)
		})
	}
}

func TestDiagnosticCallIDSurvivesPayloadOmission(t *testing.T) {
	for _, size := range []int{10, MaxPayloadBytes + 1} {
		input := map[string]any{"callId": "stable-call", "result": strings.Repeat("x", size)}
		snap, err := captureDiagnostic(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, "stable-call", input["callId"], "recording must not mutate a caller's reusable metadata")
		encoded, err := json.Marshal(snap)
		require.NoError(t, err)
		var restored models.Snapshot
		require.NoError(t, json.Unmarshal(encoded, &restored))
		require.Equal(t, "stable-call", restored.CallID)
		if size > MaxPayloadBytes {
			require.False(t, restored.Complete)
			require.Equal(t, "payload_limit", restored.Reason)
			require.JSONEq(t, `null`, string(restored.Data))
		} else {
			require.True(t, restored.Complete)
			require.JSONEq(t, `{"result":"xxxxxxxxxx"}`, string(restored.Data))
		}
	}
	snap, err := captureDiagnostic(credentials.WithSecrets(t.Context(), "private-call-token"), map[string]any{"callId": "private-call-token", "result": "ok"})
	require.NoError(t, err)
	encoded, err := json.Marshal(snap)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "private-call-token")
}

func TestCaptureCredentialExclusionAndExplicitBounds(t *testing.T) {
	ctx := credentials.WithSecrets(t.Context(), "opaque-current-access", "provider-private-key")
	input := map[string]any{
		"Authorization": "Bearer signed-access", "api_key": "private-key", "databasePassword": "private-password",
		"nested":  []any{map[string]any{"text": "opaque-current-access provider-private-key eyJabcd.efgh.ijkl api_key=another-key"}},
		"summary": "Provider-returned reasoning summary",
	}
	snap, err := CaptureSnapshot(ctx, input, true)
	require.NoError(t, err)
	for _, secret := range []string{"signed-access", "private-key", "private-password", "opaque-current-access", "provider-private-key", "eyJabcd.efgh.ijkl", "another-key"} {
		require.NotContains(t, string(snap.Data), secret)
	}
	require.Contains(t, string(snap.Data), "Provider-returned reasoning summary")
	large := strings.Repeat("x", MaxPayloadBytes+1)
	snap, err = CaptureSnapshot(ctx, large, false)
	require.NoError(t, err)
	require.False(t, snap.Complete)
	require.Equal(t, "payload_limit", snap.Reason)
	require.Equal(t, 0, snap.RetainedBytes)
	require.Greater(t, snap.OriginalBytes, MaxPayloadBytes)
	_, err = CaptureSnapshot(ctx, large, true)
	require.ErrorIs(t, err, ErrPayload)
	signature := []byte{1, 42, 0, 255, 7}
	parts := []*genai.Part{{Thought: true, Text: "signed thought", ThoughtSignature: signature}, {Text: "answer"}}
	snap, err = CaptureSnapshot(ctx, parts, true)
	require.NoError(t, err)
	var restored []*genai.Part
	require.NoError(t, json.Unmarshal(snap.Data, &restored))
	require.True(t, bytes.Equal(signature, restored[0].ThoughtSignature))
	require.Equal(t, "signed thought", restored[0].Text)
	require.Equal(t, "answer", restored[1].Text)
}

func TestRequestTokenIsRedactedFromPersistedContent(t *testing.T) {
	ctx := credentials.WithAccessToken(t.Context(), "opaque-transport-credential")
	snap, err := CaptureSnapshot(ctx, map[string]any{"text": "echo opaque-transport-credential"}, true)
	require.NoError(t, err)
	require.NotContains(t, string(snap.Data), "opaque-transport-credential")
	require.Contains(t, string(snap.Data), "[REDACTED]")
}
