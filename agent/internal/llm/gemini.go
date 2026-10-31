package llm

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/genai"
)

// Factory constructs an injectable model for a skill without running inference.
type Factory func(ctx context.Context, skillName, modelName string) (model.LLM, error)

// NewGeminiFactory constructs models without inference. Credentials are supplied by
// the caller, never a skill. Provider errors do not trigger model replacement.
func NewGeminiFactory(config genai.ClientConfig) Factory {
	return func(ctx context.Context, _, modelName string) (model.LLM, error) {
		if strings.TrimSpace(modelName) == "" {
			return nil, fmt.Errorf("model name required")
		}
		return gemini.NewModel(ctx, modelName, &config)
	}
}
