package engine

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

const maxVisibleOutputBytes = 16 << 10

// RunTurn executes one ADK turn and projects visible output with version metadata.
// The chat service supplies authenticated identity, admission and a run budget.
func (engine *Engine) RunTurn(
	ctx context.Context,
	key models.Key,
	request models.ChatRequest,
	output models.Output,
	emit func(models.Output) error,
) error {
	ctx = withRunLogger(ctx, engine.logger, request)
	userContent := genai.NewContentFromText(request.Text, "user")
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeSSE,
	}
	if key.Session == "" {
		return &models.Error{Code: models.Internal}
	}
	events := engine.runner.Run(
		ctx, key.Owner, key.Session, userContent, runConfig,
	)

	partialByAgent := map[string]bool{}
	hasVisibleText := false
	outputBytes := 0

	for event, err := range events {
		if err != nil {
			return &models.Error{Code: models.PublicCode(err)}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if event == nil {
			continue
		}
		if event.ErrorCode != "" {
			return &models.Error{Code: models.Unavailable}
		}
		engine.logHandoff(ctx, event)
		engine.logHandback(ctx, event)
		if event.Content == nil || event.Content.Role != "model" {
			continue
		}

		// ADK emits partial deltas followed by a complete copy.
		if !event.Partial && partialByAgent[event.Author] {
			partialByAgent[event.Author] = false
			continue
		}

		delta := extractVisibleText(event)
		if delta == "" {
			continue
		}
		if event.Partial {
			partialByAgent[event.Author] = true
		}

		outputBytes += len(delta)
		if outputBytes > maxVisibleOutputBytes {
			return models.ErrBudget
		}

		metadata, ok := engine.agentMetadata[event.Author]
		if !ok {
			return &models.Error{Code: models.Internal}
		}
		output.Kind = "text"
		output.Delta = delta
		output.InvocationID = event.InvocationID
		output.Skill = metadata.Name
		output.SkillVersion = metadata.Version
		output.Model = metadata.ModelSelection.Selected
		output.ProviderModelVersion = event.ModelVersion

		if err := emit(output); err != nil {
			return err
		}
		hasVisibleText = true
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if !hasVisibleText {
		return errors.New("agent returned no visible response")
	}
	return nil
}

func extractVisibleText(event *session.Event) string {
	var text strings.Builder
	for _, part := range event.Content.Parts {
		if part == nil || part.Thought {
			continue
		}
		if part.FunctionCall != nil || part.FunctionResponse != nil {
			continue
		}
		text.WriteString(part.Text)
	}
	return text.String()
}

type runLoggerKey struct{}

func withRunLogger(ctx context.Context, logger *slog.Logger, request models.ChatRequest) context.Context {
	logger = logger.With(
		"conversationId", request.ConversationID,
		"messageId", request.MessageID,
		"runId", request.RunID,
	)
	return context.WithValue(ctx, runLoggerKey{}, logger)
}

func (engine *Engine) runLogger(ctx context.Context) *slog.Logger {
	if logger, exists := ctx.Value(runLoggerKey{}).(*slog.Logger); exists {
		return logger
	}
	return engine.logger
}

// Native task delegation bypasses ordinary tool callbacks. Observe its emitted
// function call instead, recording only the selected registered leaf and IDs.
func (engine *Engine) logHandoff(ctx context.Context, event *session.Event) {
	if event.Partial || event.Author != "orchestrator" || event.Content == nil {
		return
	}
	for _, part := range event.Content.Parts {
		if part == nil || part.FunctionCall == nil {
			continue
		}
		call := part.FunctionCall
		target, registered := engine.agentMetadata[call.Name]
		if registered && target.Name != "orchestrator" {
			engine.runLogger(ctx).InfoContext(ctx, "agent handoff",
				"from", "orchestrator",
				"to", target.Name,
				"invocationId", event.InvocationID,
				"callId", call.ID,
			)
		}
	}
}

// ADK synthesizes a user-role function response only after a delegated task
// completes. A clarification ends a run without producing this handback event.
func (engine *Engine) logHandback(ctx context.Context, event *session.Event) {
	if event.Partial || event.Content == nil || event.Content.Role != "user" {
		return
	}
	for _, part := range event.Content.Parts {
		if part == nil || part.FunctionResponse == nil {
			continue
		}
		response := part.FunctionResponse
		source, registered := engine.agentMetadata[response.Name]
		if !registered || source.Name == "orchestrator" {
			continue
		}
		engine.runLogger(ctx).InfoContext(ctx, "agent handback",
			"from", source.Name,
			"to", "orchestrator",
			"invocationId", event.InvocationID,
			"callId", response.ID,
		)
	}
}
