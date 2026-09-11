package observability

import (
	"io"
	"log/slog"
)

const releaseMode = "release"

// NewLogger creates the application logger. Production uses line-delimited JSON
// while local modes keep the same keys in a more readable format.
func NewLogger(mode string, output io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, attribute slog.Attr) slog.Attr {
			switch attribute.Key {
			case slog.TimeKey:
				attribute.Key = "timestamp"
			case slog.MessageKey:
				attribute.Key = "event"
			}
			return attribute
		},
	}

	if mode == releaseMode {
		return slog.New(slog.NewJSONHandler(output, options))
	}
	return slog.New(slog.NewTextHandler(output, options))
}
