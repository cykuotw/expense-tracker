package updatestate

import (
	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
)

// ADK runs calls in one response concurrently and merges their state deltas.
// Require state edits alone to preserve whole-value replacement/unset semantics.
func ValidateModelResponse(_ agent.Context, response *model.LLMResponse, responseErr error) (*model.LLMResponse, error) {
	if responseErr != nil || response == nil || response.Content == nil {
		return nil, responseErr
	}
	calls, updates := 0, 0
	for _, part := range response.Content.Parts {
		if part != nil && part.FunctionCall != nil {
			calls++
			if part.FunctionCall.Name == ToolName {
				updates++
			}
		}
	}
	if updates > 0 && calls > 1 {
		return nil, &models.Error{Code: models.Validation}
	}
	return nil, nil
}
