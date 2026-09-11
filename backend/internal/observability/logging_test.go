package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggerUsesCanonicalKeys(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "production JSON", mode: releaseMode},
		{name: "local text", mode: "debug"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := NewLogger(test.mode, &output)

			logger.Info("test_event", slog.String("request_id", "request-1"))

			line := output.String()
			assert.Contains(t, line, "timestamp")
			assert.Contains(t, line, "level")
			assert.Contains(t, line, "event")
			assert.Contains(t, line, "request_id")
			assert.NotContains(t, line, "msg")

			if test.mode == releaseMode {
				var entry map[string]any
				require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
				assert.Equal(t, "test_event", entry["event"])
				assert.Equal(t, "request-1", entry["request_id"])
			}
		})
	}
}

func decodeLogEntries(t *testing.T, output string) []map[string]any {
	t.Helper()

	entries := make([]map[string]any, 0)
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		entries = append(entries, entry)
	}
	return entries
}
