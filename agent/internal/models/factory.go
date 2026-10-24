package models

import (
	"context"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/genai"
)

type Factory func(ctx context.Context, skillName, modelName string) (model.LLM, error)

// GeminiFactory constructs models without inference. Credentials are supplied by
// the caller, never a skill. Provider errors do not trigger model replacement.
func GeminiFactory(config genai.ClientConfig) Factory {
	return func(ctx context.Context, _, modelName string) (model.LLM, error) {
		if _, err := Resolve(modelName, modelName); err != nil {
			return nil, err
		}
		return gemini.NewModel(ctx, modelName, &config)
	}
}
