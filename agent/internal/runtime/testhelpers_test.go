package runtime

import (
	"context"
	"errors"
	"io/fs"
	"iter"
	"strings"
	"testing"
	"testing/fstest"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/skills"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

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

type unavailableBalance struct{}

func (unavailableBalance) GetBalance(context.Context, string) (tools.BalanceResult, error) {
	return tools.BalanceResult{}, errors.New("private backend error")
}

func bundledFiles(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(skills.Files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, err := fs.ReadFile(skills.Files, path)
			if err != nil {
				return err
			}
			files[path] = &fstest.MapFile{Data: data}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func runMessage(
	t *testing.T,
	app *App,
	conversationID string,
	messageID string,
	text string,
) ([]Output, string, error) {
	t.Helper()
	var outputs []Output
	var answer strings.Builder
	err := app.Run(t.Context(), Request{
		ConversationID: conversationID,
		MessageID:      messageID,
		Text:           text,
	}, func(output Output) error {
		outputs = append(outputs, output)
		answer.WriteString(output.Delta)
		return nil
	})
	return outputs, answer.String(), err
}

// routingObservation records only the coordinator's declared capabilities.
type routingObservation struct {
	calls                int
	sawEchoDescription   bool
	receivedBusinessTool bool
}

func (observation *routingObservation) modelFactory(
	ctx context.Context,
	skillName string,
	modelName string,
) (model.LLM, error) {
	provider, err := models.FixtureFactory(ctx, skillName, modelName)
	if err != nil {
		return nil, err
	}
	return modelFunc(func(
		ctx context.Context,
		request *model.LLMRequest,
		stream bool,
	) iter.Seq2[*model.LLMResponse, error] {
		if skillName == "orchestrator" {
			observation.calls++
			observation.inspect(request)
		}
		return provider.GenerateContent(ctx, request, stream)
	}), nil
}

func (observation *routingObservation) inspect(request *model.LLMRequest) {
	for _, tool := range request.Config.Tools {
		for _, function := range tool.FunctionDeclarations {
			if function.Name == "get_balance" {
				observation.receivedBusinessTool = true
			}
			isEcho := function.Name == "skill_echo"
			if isEcho && strings.Contains(function.Description, "synthetic message") {
				observation.sawEchoDescription = true
			}
		}
	}
}

func failingModelFactory(scenario string) models.Factory {
	return func(ctx context.Context, skillName, modelName string) (model.LLM, error) {
		if scenario == "premature-finish" && skillName == "orchestrator" {
			return models.FixtureFactory(ctx, skillName, modelName)
		}
		return modelFunc(func(
			context.Context,
			*model.LLMRequest,
			bool,
		) iter.Seq2[*model.LLMResponse, error] {
			return func(yield func(*model.LLMResponse, error) bool) {
				if scenario == "provider" {
					yield(nil, errors.New("private provider failure"))
					return
				}

				functionName := "unknown_agent"
				arguments := map[string]any{"request": "balance demo"}
				if scenario == "premature-finish" {
					functionName = "finish_task"
					arguments = map[string]any{"result": "fabricated balance"}
				}
				yield(&model.LLMResponse{
					Content: &genai.Content{
						Role: "model",
						Parts: []*genai.Part{
							{
								FunctionCall: &genai.FunctionCall{
									Name: functionName,
									Args: arguments,
								},
							},
						},
					},
				}, nil)
			}
		}), nil
	}
}
