package runtime

import (
	"context"
	"errors"
	"strings"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

const maxVisibleOutputBytes = 16 << 10

type ownerKey struct{}

// execute projects ADK responses into visible output, retaining correlation and
// version metadata while hiding internal events and duplicate final copies.
func (app *App) execute(
	ctx context.Context,
	request Request,
	output Output,
	emit func(Output) error,
) error {
	userContent := genai.NewContentFromText(request.Text, "user")
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeSSE,
	}
	owner, _ := ctx.Value(ownerKey{}).(string)
	events := app.runner.Run(
		ctx, owner, request.ConversationID, userContent, runConfig,
	)

	partialByAgent := map[string]bool{}
	hasVisibleText := false
	outputBytes := 0

	for event, err := range events {
		if err != nil {
			return &tools.Error{Code: tools.PublicCode(err)}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if event == nil {
			continue
		}
		if event.ErrorCode != "" {
			return &tools.Error{Code: tools.Unavailable}
		}
		app.logHandoff(ctx, event)
		app.logHandback(ctx, event)
		if event.Content == nil || event.Content.Role != "model" {
			continue
		}

		// ADK emits partial deltas followed by a complete copy.
		if !event.Partial && partialByAgent[event.Author] {
			partialByAgent[event.Author] = false
			continue
		}

		delta := visibleText(event)
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

		skillName, ok := app.agentSkills[event.Author]
		if !ok {
			return &tools.Error{Code: tools.Internal}
		}
		diagnostic := app.diagnostics[skillName]
		output.Kind = "text"
		output.Delta = delta
		output.InvocationID = event.InvocationID
		output.Skill = skillName
		output.SkillVersion = diagnostic.SkillVersion
		output.Model = diagnostic.Model.Resolved
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

func visibleText(event *session.Event) string {
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
