package ocr

import (
	"encoding/json"
	"log/slog"
	"time"
)

// GeminiTokenUsage contains only counts, never receipt or generated content.
type GeminiTokenUsage struct {
	Input, Output, Thinking, CachedInput, ToolInput, Total int64
	standardTier                                           bool
}

type GeminiUsageObservation struct {
	Model, Outcome, UsageStatus, CostStatus string
	HTTPStatus                              int
	LatencyMillis                           int64
	Usage                                   *GeminiTokenUsage
	EstimatedCostUSD                        *float64
	PricingVersion                          string
}

// LogAttrs is an explicit allowlist. Do not add raw requests/responses here.
func (o GeminiUsageObservation) LogAttrs() []slog.Attr {
	attributes := []slog.Attr{
		slog.String("provider", "gemini"), slog.String("model", o.Model),
		slog.String("outcome", o.Outcome), slog.Int("http_status", o.HTTPStatus),
		slog.Int64("provider_latency_ms", o.LatencyMillis),
		slog.String("usage_status", o.UsageStatus), slog.String("cost_status", o.CostStatus),
	}
	if u := o.Usage; u != nil {
		attributes = append(attributes,
			slog.Int64("input_tokens", u.Input), slog.Int64("output_tokens", u.Output),
			slog.Int64("thinking_tokens", u.Thinking), slog.Int64("cached_input_tokens", u.CachedInput),
			slog.Int64("tool_input_tokens", u.ToolInput), slog.Int64("total_tokens", u.Total),
		)
	}
	if o.EstimatedCostUSD != nil {
		attributes = append(attributes,
			slog.Float64("estimated_cost_usd", *o.EstimatedCostUSD),
			slog.String("cost_basis", "standard_paid_tier"),
			slog.String("pricing_version", o.PricingVersion),
		)
	}
	return attributes
}

func parseGeminiUsage(raw []byte) (*GeminiTokenUsage, string) {
	var envelope struct {
		Usage *struct {
			Input       *int64 `json:"promptTokenCount"`
			Output      int64  `json:"candidatesTokenCount"`
			Thinking    int64  `json:"thoughtsTokenCount"`
			CachedInput int64  `json:"cachedContentTokenCount"`
			ToolInput   int64  `json:"toolUsePromptTokenCount"`
			Total       *int64 `json:"totalTokenCount"`
			ServiceTier string `json:"serviceTier"`
		} `json:"usageMetadata"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return nil, "invalid"
	}
	u := envelope.Usage
	if u == nil {
		return nil, "missing"
	}
	if u.Input == nil || u.Total == nil {
		return nil, "invalid"
	}
	// Bounds prevent overflow and reject invalid/inconsistent provider telemetry.
	for _, count := range []int64{*u.Input, u.Output, u.Thinking, u.CachedInput, u.ToolInput, *u.Total} {
		if count < 0 || count > 1_000_000_000 {
			return nil, "invalid"
		}
	}
	if u.CachedInput > *u.Input || *u.Input+u.Output+u.Thinking+u.ToolInput != *u.Total {
		return nil, "invalid"
	}
	return &GeminiTokenUsage{
		Input: *u.Input, Output: u.Output, Thinking: u.Thinking,
		CachedInput: u.CachedInput, ToolInput: u.ToolInput, Total: *u.Total,
		standardTier: u.ServiceTier == "" || u.ServiceTier == "unspecified" || u.ServiceTier == "standard",
	}, "available"
}

func (o *GeminiUsageObservation) estimateCost(now time.Time) {
	o.CostStatus = "usage_unavailable"
	if o.Usage == nil {
		return
	}
	o.CostStatus = "pricing_unavailable"
	if o.Model != "gemini-3.8-flash" {
		return
	}
	o.CostStatus = "unsupported_usage"
	if !o.Usage.standardTier || o.Usage.ToolInput != 0 {
		return
	}
	// Standard synchronous text/image rates, verified 2026-10-20:
	// https://ai.google.dev/gemini-api/docs/pricing#gemini-3.8-flash
	// Per-token nano-USD keeps arithmetic exact before numeric log serialization.
	inputRate, outputRate, cacheRate := int64(750), int64(3750), int64(75)
	o.PricingVersion = "google-standard-2026-10-20-intro"
	if !now.Before(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		inputRate, outputRate, cacheRate = 1500, 7500, 150
		o.PricingVersion = "google-standard-2026-10-20-from-2027"
	}
	u := o.Usage
	nanoUSD := (u.Input-u.CachedInput)*inputRate + u.CachedInput*cacheRate + (u.Output+u.Thinking)*outputRate
	cost := float64(nanoUSD) / 1_000_000_000
	o.EstimatedCostUSD = &cost
	o.CostStatus = "estimated"
}
