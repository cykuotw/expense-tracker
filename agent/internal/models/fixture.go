package models

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"iter"
	"strings"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

// FixtureFactory exercises ADK's native delegation/tools without inference.
// It recognizes balance requests and explicit leaf skill names deterministically.
func FixtureFactory(_ context.Context, skillName, modelName string) (model.LLM, error) {
	return &fixtureModel{
		skill: skillName,
		name:  modelName,
	}, nil
}

type fixtureModel struct {
	skill string
	name  string
}

func (fixture *fixtureModel) Name() string {
	return fixture.name
}

func (fixture *fixtureModel) GenerateContent(
	ctx context.Context,
	request *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(nil, err)
			return
		}

		text, result := latestInput(request.Contents)
		switch fixture.skill {
		case "orchestrator":
			fixture.respondAsRouter(request, text, result, stream, yield)
		case "get-balance":
			respondWithBalance(text, result, yield)
		default:
			// Other leaves exercise discovery without implementing domain logic.
			summary := fmt.Sprintf("Synthetic fixture %s: %s", fixture.skill, text)
			yield(functionCallResponse("finish_task", map[string]any{
				"result": summary,
			}), nil)
		}
	}
}

func (fixture *fixtureModel) respondAsRouter(
	request *model.LLMRequest,
	text string,
	result *genai.FunctionResponse,
	stream bool,
	yield func(*model.LLMResponse, error) bool,
) {
	if result != nil {
		answer, err := delegatedAnswer(result)
		if err != nil {
			yield(nil, err)
			return
		}
		prefix := "Synthetic fixture result: "
		if result.Name == "skill_get_balance" {
			prefix = "Synthetic fixture balance: "
		}
		streamText(prefix, answer, stream, yield)
		return
	}

	delegate := selectFixtureDelegate(request, text)
	if delegate != "" {
		yield(functionCallResponse(delegate, map[string]any{"request": text}), nil)
		return
	}

	streamText(
		"", "I can check group balances. Please include the group ID.",
		stream, yield,
	)
}

func delegatedAnswer(result *genai.FunctionResponse) (string, error) {
	body := result.Response["result"]
	if body == nil {
		body = result.Response["output"]
	}
	if answer, isText := body.(string); isText {
		return answer, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func selectFixtureDelegate(request *model.LLMRequest, text string) string {
	lowerText := strings.ToLower(text)
	isBalanceRequest := strings.Contains(lowerText, "balance") ||
		strings.Contains(lowerText, "餘額")

	for _, declaredTool := range request.Config.Tools {
		for _, function := range declaredTool.FunctionDeclarations {
			if !strings.HasPrefix(function.Name, "skill_") {
				continue
			}
			skillName := strings.TrimPrefix(function.Name, "skill_")
			skillName = strings.ReplaceAll(skillName, "_", "-")

			balanceMatches := skillName == "get-balance" && isBalanceRequest
			if balanceMatches || strings.Contains(lowerText, skillName) {
				return function.Name
			}
		}
	}
	return ""
}

func respondWithBalance(
	text string,
	result *genai.FunctionResponse,
	yield func(*model.LLMResponse, error) bool,
) {
	if result != nil && result.Name == "get_balance" {
		payload, err := json.Marshal(result.Response)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(functionCallResponse("finish_task", map[string]any{
			"result": string(payload),
		}), nil)
		return
	}
	if !strings.Contains(strings.ToLower(text), "demo") {
		yield(&model.LLMResponse{
			Content:      genai.NewContentFromText("Which group should I check?", "model"),
			TurnComplete: true,
		}, nil)
		return
	}
	yield(functionCallResponse("get_balance", map[string]any{"groupId": "demo"}), nil)
}

func functionCallResponse(name string, arguments map[string]any) *model.LLMResponse {
	return &model.LLMResponse{
		Content: &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{
					FunctionCall: &genai.FunctionCall{
						Name: name,
						Args: arguments,
					},
				},
			},
		},
	}
}

// Ignore old tool results once a newer user message starts another run.
func latestInput(contents []*genai.Content) (string, *genai.FunctionResponse) {
	var text string
	var result *genai.FunctionResponse
	for _, content := range contents {
		if content == nil {
			continue
		}
		for _, part := range content.Parts {
			if part == nil {
				continue
			}
			if content.Role == "user" && part.Text != "" {
				text = part.Text
				result = nil
			}
			if part.FunctionResponse != nil {
				result = part.FunctionResponse
			}
		}
	}
	return text, result
}

func streamText(
	prefix string,
	body string,
	stream bool,
	yield func(*model.LLMResponse, error) bool,
) {
	if stream {
		for _, delta := range []string{prefix, body} {
			if delta == "" {
				continue
			}
			response := &model.LLMResponse{
				Content: genai.NewContentFromText(delta, "model"),
				Partial: true,
			}
			if !yield(response, nil) {
				return
			}
		}
	}
	yield(&model.LLMResponse{
		Content:      genai.NewContentFromText(prefix+body, "model"),
		TurnComplete: true,
	}, nil)
}
