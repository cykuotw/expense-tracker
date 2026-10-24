package runtime

import (
	"context"
	"log/slog"

	"google.golang.org/adk/v2/session"
)

type runLoggerKey struct{}

func withRunLogger(ctx context.Context, logger *slog.Logger, request Request, runID string) context.Context {
	logger = logger.With(
		"conversationId", request.ConversationID,
		"messageId", request.MessageID,
		"runId", runID,
	)
	return context.WithValue(ctx, runLoggerKey{}, logger)
}

func (app *App) runLogger(ctx context.Context) *slog.Logger {
	if logger, exists := ctx.Value(runLoggerKey{}).(*slog.Logger); exists {
		return logger
	}
	return app.logger
}

// Native task delegation bypasses ordinary tool callbacks. Observe its emitted
// function call instead, recording only the selected registered leaf and IDs.
func (app *App) logHandoff(ctx context.Context, event *session.Event) {
	if event.Partial || event.Author != "orchestrator" || event.Content == nil {
		return
	}
	for _, part := range event.Content.Parts {
		if part == nil || part.FunctionCall == nil {
			continue
		}
		call := part.FunctionCall
		target, registered := app.agentSkills[call.Name]
		if registered && target != "orchestrator" {
			app.runLogger(ctx).InfoContext(ctx, "agent handoff",
				"from", "orchestrator",
				"to", target,
				"invocationId", event.InvocationID,
				"callId", call.ID,
			)
		}
	}
}

// ADK synthesizes a user-role function response only after a delegated task
// completes. A clarification ends a run without producing this handback event.
func (app *App) logHandback(ctx context.Context, event *session.Event) {
	if event.Partial || event.Content == nil || event.Content.Role != "user" {
		return
	}
	for _, part := range event.Content.Parts {
		if part == nil || part.FunctionResponse == nil {
			continue
		}
		response := part.FunctionResponse
		source, registered := app.agentSkills[response.Name]
		if !registered || source == "orchestrator" {
			continue
		}
		app.runLogger(ctx).InfoContext(ctx, "agent handback",
			"from", source,
			"to", "orchestrator",
			"invocationId", event.InvocationID,
			"callId", response.ID,
		)
	}
}
