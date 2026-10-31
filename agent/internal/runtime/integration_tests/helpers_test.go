package integration_test

import (
	"context"
	"iter"
	"strings"
	"testing"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
)

const sessionAppName = "expense-agent"

type modelFunc func(context.Context, *model.LLMRequest, bool) iter.Seq2[*model.LLMResponse, error]

func (modelFunc) Name() string {
	return "fixture"
}
func (generate modelFunc) GenerateContent(
	ctx context.Context,
	request *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return generate(ctx, request, stream)
}

func runMessage(
	t *testing.T,
	app *runtime.App,
	conversationID string,
	messageID string,
	text string,
) ([]models.Output, string, error) {
	t.Helper()
	var outputs []models.Output
	var answer strings.Builder
	err := app.Run(t.Context(), models.ChatRequest{
		ConversationID: conversationID,
		MessageID:      messageID,
		Text:           text,
	}, func(output models.Output) error {
		outputs = append(outputs, output)
		answer.WriteString(output.Delta)
		return nil
	})
	return outputs, answer.String(), err
}

func testCatalog(t *testing.T, client getbalance.Client) map[string]tool.Tool {
	t.Helper()
	catalog, err := tools.NewCatalog(client)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
