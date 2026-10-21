package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/internal/observability"
	"github.com/stretchr/testify/require"
)

const sampleUsage = `{"promptTokenCount":1000,"cachedContentTokenCount":200,"candidatesTokenCount":100,"thoughtsTokenCount":50,"totalTokenCount":1150}`

type usageTestTransport func(*http.Request) (*http.Response, error)

func (f usageTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGeminiUsageLoggingIncludesRejectedOutputAndNeverReceiptContent(t *testing.T) {
	encoded, err := json.Marshal(sampleExtraction)
	require.NoError(t, err)
	response := func(output, finish, usage string) string {
		return `{"candidates":[{"content":{"parts":[{"text":` + output + `}]},"finishReason":"` + finish + `"}],"usageMetadata":` + usage + `,"modelVersion":"private-provider-content"}`
	}
	for _, test := range []struct {
		name, body, outcome, usageStatus, costStatus string
		status                                       int
		transportError                               bool
		wantError                                    bool
	}{
		{"success", response(string(encoded), "STOP", sampleUsage), "success", "available", "estimated", 200, false, false},
		{"token limit", response(string(encoded), "MAX_TOKENS", sampleUsage), "response_rejected", "available", "estimated", 200, false, true},
		{"invalid extraction", response(`"private-receipt-content"`, "STOP", sampleUsage), "response_rejected", "available", "estimated", 200, false, true},
		{"missing usage", response(string(encoded), "STOP", "null"), "success", "missing", "usage_unavailable", 200, false, false},
		{"invalid usage", response(string(encoded), "STOP", `{"promptTokenCount":100,"totalTokenCount":1}`), "success", "invalid", "usage_unavailable", 200, false, false},
		{"throttled", `{"error":"private-receipt-content"}`, "http_error", "unavailable", "usage_unavailable", 429, false, true},
		{"bad request", `{"error":{"status":"INVALID_ARGUMENT","message":"private-api-key private-receipt-content","details":[{"reason":"API_KEY_INVALID","metadata":{"key":"private-api-key"}}]}}`, "http_error", "unavailable", "usage_unavailable", 400, false, true},
		{"network", "", "request_failed", "unavailable", "usage_unavailable", 0, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider, err := NewGeminiProvider("private-api-key", DefaultGeminiModel)
			require.NoError(t, err)
			provider.client = &http.Client{Transport: usageTestTransport(func(r *http.Request) (*http.Response, error) {
				if test.transportError {
					return nil, errors.New("private-api-key private-receipt-content")
				}
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body)), Header: make(http.Header)}, nil
			})}
			var logs bytes.Buffer
			logger := observability.NewLogger("release", &logs)
			observations := 0
			provider.SetUsageObserver(func(ctx context.Context, o GeminiUsageObservation) {
				observations++
				logger.LogAttrs(ctx, slog.LevelInfo, "ocr_gemini_usage", o.LogAttrs()...)
			})
			result, err := provider.AnalyzeExpense(t.Context(), []byte("private-image-bytes"))
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, observations)
			var event map[string]any
			require.NoError(t, json.Unmarshal(logs.Bytes(), &event))
			require.Equal(t, "ocr_gemini_usage", event["event"])
			require.Equal(t, test.outcome, event["outcome"])
			require.Equal(t, test.usageStatus, event["usage_status"])
			require.Equal(t, test.costStatus, event["cost_status"])
			if test.name == "bad request" {
				require.Equal(t, float64(400), event["http_status"])
				require.Equal(t, "INVALID_ARGUMENT", event["provider_error_status"])
				require.Equal(t, "API_KEY_INVALID", event["provider_error_reason"])
			}
			if test.costStatus == "estimated" {
				require.Equal(t, float64(1000), event["input_tokens"])
				require.Equal(t, float64(100), event["output_tokens"])
				require.Equal(t, float64(50), event["thinking_tokens"])
				require.Equal(t, float64(200), event["cached_input_tokens"])
				require.Equal(t, float64(1150), event["total_tokens"])
				require.Greater(t, event["estimated_cost_usd"].(float64), 0.0)
			} else {
				require.NotContains(t, event, "estimated_cost_usd")
				require.NotContains(t, event, "input_tokens")
			}
			if !test.wantError {
				require.NotNil(t, result.Usage)
				require.Equal(t, DefaultGeminiModel, result.Usage.Model)
				if test.costStatus == "estimated" {
					require.Equal(t, event["estimated_cost_usd"], *result.Usage.EstimatedCostUSD)
					require.Equal(t, int64(1000), result.Usage.Tokens.Input)
				} else {
					require.Nil(t, result.Usage.Tokens)
					require.Nil(t, result.Usage.EstimatedCostUSD)
				}
				raw, marshalErr := json.Marshal(result)
				require.NoError(t, marshalErr)
				require.NotContains(t, string(raw), "estimatedCostUsd")
			}
			for _, private := range []string{"private-api-key", "private-receipt-content", "private-provider-content", "private-image-bytes", "Shoppers Drug Mart", "NEILSON MILK", GeminiPrompt} {
				require.NotContains(t, logs.String(), private)
			}
			_, _, _ = ReplayGeminiResponse([]byte(test.body))
			require.Equal(t, 1, observations, "offline replay must not emit billable usage")
		})
	}
}

func TestGeminiErrorCategoryIsBoundedAndAllowlisted(t *testing.T) {
	for _, test := range []struct{ body, status, reason string }{
		{`{"error":{"status":"FAILED_PRECONDITION","message":"private receipt"}}`, "FAILED_PRECONDITION", ""},
		{`{"error":{"status":"private receipt","details":[{"reason":"private-key"}]}}`, "UNKNOWN", ""},
		{`{"error":{"status":"PERMISSION_DENIED","details":[{"reason":"SERVICE_DISABLED"}]}}`, "PERMISSION_DENIED", "SERVICE_DISABLED"},
		{`not json`, "UNKNOWN", ""},
		{strings.Repeat("x", 16*1024+1), "UNKNOWN", ""},
	} {
		status, reason := geminiErrorCategory(strings.NewReader(test.body))
		require.Equal(t, test.status, status)
		require.Equal(t, test.reason, reason)
	}
}

func TestGeminiUsageCostAccountsForCacheThinkingAndPriceTransition(t *testing.T) {
	u, status := parseGeminiUsage([]byte(`{"usageMetadata":` + sampleUsage + `}`))
	require.Equal(t, "available", status)
	for _, test := range []struct {
		at      string
		cost    float64
		version string
	}{
		{"2026-12-31T23:59:59Z", 0.0011775, "google-standard-2026-10-20-intro"},
		{"2027-01-01T00:00:00Z", 0.002355, "google-standard-2026-10-20-from-2027"},
	} {
		at, err := time.Parse(time.RFC3339, test.at)
		require.NoError(t, err)
		o := GeminiUsageObservation{Model: DefaultGeminiModel, Usage: u}
		o.estimateCost(at)
		require.Equal(t, "estimated", o.CostStatus)
		require.InDelta(t, test.cost, *o.EstimatedCostUSD, 1e-12)
		require.Equal(t, test.version, o.PricingVersion)
	}
	for _, test := range []struct{ model, usage, status string }{
		{"gemini-unknown", sampleUsage, "pricing_unavailable"},
		{DefaultGeminiModel, strings.Replace(sampleUsage, `"totalTokenCount":1150`, `"totalTokenCount":1150,"serviceTier":"priority"`, 1), "unsupported_usage"},
		{DefaultGeminiModel, strings.Replace(sampleUsage, `"totalTokenCount":1150`, `"totalTokenCount":1151,"toolUsePromptTokenCount":1`, 1), "unsupported_usage"},
	} {
		u, _ := parseGeminiUsage([]byte(`{"usageMetadata":` + test.usage + `}`))
		o := GeminiUsageObservation{Model: test.model, Usage: u}
		o.estimateCost(time.Now())
		require.Equal(t, test.status, o.CostStatus)
		require.Nil(t, o.EstimatedCostUSD)
	}
}

func TestGeminiUsageRejectsMisleadingTelemetry(t *testing.T) {
	for _, raw := range []string{
		`{}`, `null`, `{"promptTokenCount":100}`, `{"totalTokenCount":100}`,
		`{"promptTokenCount":-1,"totalTokenCount":-1}`,
		`{"promptTokenCount":1,"cachedContentTokenCount":2,"totalTokenCount":1}`,
		`{"promptTokenCount":1,"candidatesTokenCount":-1,"totalTokenCount":0}`,
		`{"promptTokenCount":1,"thoughtsTokenCount":1,"totalTokenCount":1}`,
		`{"promptTokenCount":"100","totalTokenCount":100}`,
		`{"promptTokenCount":1000000001,"totalTokenCount":1000000001}`,
		`{"promptTokenCount":1e100,"totalTokenCount":1e100}`,
	} {
		u, status := parseGeminiUsage([]byte(`{"usageMetadata":` + raw + `}`))
		require.Nil(t, u, raw)
		require.NotEqual(t, "available", status, raw)
	}
	// Proto JSON omits zero output/thinking counts for blocked prompts.
	u, status := parseGeminiUsage([]byte(`{"usageMetadata":{"promptTokenCount":100,"totalTokenCount":100}}`))
	require.Equal(t, "available", status)
	require.Zero(t, u.Output)
	require.Zero(t, u.Thinking)
}
