package fixtures

import "expense-tracker/agent/internal/models"

func ModelPolicy() models.ModelPolicy {
	return models.ModelPolicy{
		Default: models.ModelProfile{Name: "fixture-default", MaxOutputTokens: 2048},
		Router:  models.ModelProfile{Name: "fixture-router", MaxOutputTokens: 2048},
	}
}

func Limits() models.LLMLimits {
	return models.LLMLimits{MaxModelCalls: 6, MaxToolCalls: 2, MaxStateUpdates: 4,
		MaxContextBytes: 64 << 10, MaxResponseBytes: 32 << 10,
		MaxTotalResponseBytes: 256 << 10, MaxResponseEvents: 256}
}
