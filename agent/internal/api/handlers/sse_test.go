package handlers

import (
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"expense-tracker/agent/internal/models"

	"github.com/stretchr/testify/require"
)

func TestDurableOutputUsesStableMessageAndCursor(t *testing.T) {
	response := httptest.NewRecorder()
	stream := newChatStream(response, "conversation")
	var ids []string
	for _, kind := range []string{"started", "text", "finished"} {
		id := uuid.New().String()
		ids = append(ids, id)
		require.NoError(t, stream.emit(models.Output{Kind: kind, RunID: "durable-run", MessageID: "durable-message", EventID: id, CreatedAt: time.Now().UTC(), Delta: "visible"}))
	}
	require.Equal(t, "text/event-stream", response.Header().Get("Content-Type"))
	require.Contains(t, response.Body.String(), "id: "+ids[0])
	require.Contains(t, response.Body.String(), "id: "+ids[2])
	require.Contains(t, response.Body.String(), `"messageId":"durable-message"`)
}
