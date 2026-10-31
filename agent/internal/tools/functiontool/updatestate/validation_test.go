package updatestate

import (
	"testing"

	"expense-tracker/agent/internal/models"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

func TestStateCallsMustBeAlone(t *testing.T) {
	for _, names := range [][]string{{ToolName, ToolName}, {ToolName, "skill_get_balance"}} {
		response := &model.LLMResponse{Content: &genai.Content{}}
		for _, name := range names {
			response.Content.Parts = append(response.Content.Parts, &genai.Part{FunctionCall: &genai.FunctionCall{Name: name}})
		}
		_, err := ValidateModelResponse(nil, response, nil)
		require.Equal(t, models.Validation, models.PublicCode(err))
	}
	_, err := ValidateModelResponse(nil, &model.LLMResponse{Content: &genai.Content{Parts: []*genai.Part{{FunctionCall: &genai.FunctionCall{Name: ToolName}}}}}, nil)
	require.NoError(t, err)
}
