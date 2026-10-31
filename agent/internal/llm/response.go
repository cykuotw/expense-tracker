package llm

import (
	"encoding/json/v2"
	"slices"

	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

type responseUsage struct {
	limits models.LLMLimits
	events int64
	bytes  int64
}

func (usage *responseUsage) record(response *model.LLMResponse) error {
	payload, err := json.Marshal(response)
	usage.events++
	usage.bytes += int64(len(payload))

	overLimit := len(payload) > int(usage.limits.MaxResponseBytes) ||
		usage.bytes > int64(usage.limits.MaxTotalResponseBytes) ||
		usage.events > int64(usage.limits.MaxResponseEvents)
	if err != nil || overLimit {
		return models.ErrBudget
	}
	return nil
}

func sanitizeResponse(response *model.LLMResponse) *model.LLMResponse {
	sanitized := *response
	sanitized.CustomMetadata = nil
	if response.Content != nil {
		content := *response.Content
		content.Parts = slices.DeleteFunc(
			slices.Clone(content.Parts),
			func(part *genai.Part) bool {
				return part == nil || (part.Thought && len(part.ThoughtSignature) == 0)
			},
		)
		sanitized.Content = &content
	}
	return &sanitized
}

func hasThoughtParts(response *model.LLMResponse) bool {
	if response.Content != nil {
		for _, part := range response.Content.Parts {
			if part != nil && part.Thought {
				return true
			}
		}
	}
	return false
}
