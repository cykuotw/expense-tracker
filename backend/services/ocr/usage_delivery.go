package ocr

import (
	"context"
	"expense-tracker/backend/services/ocrusage"
	"time"
)

type usageIdentity struct{ accountID, requestID string }
type usageIdentityKey struct{}

// Only the verified capability supplies these IDs; browser usage is never trusted.
func withUsageIdentity(ctx context.Context, accountID, requestID string) context.Context {
	return context.WithValue(ctx, usageIdentityKey{}, usageIdentity{accountID, requestID})
}

func UsageRecord(ctx context.Context, o GeminiUsageObservation, now time.Time) ocrusage.Record {
	identity, _ := ctx.Value(usageIdentityKey{}).(usageIdentity)
	r := ocrusage.Record{
		RequestID: identity.requestID, AccountID: identity.accountID,
		Model: o.Model, Outcome: o.Outcome, UsageStatus: o.UsageStatus,
		CostStatus: o.CostStatus, PricingVersion: o.PricingVersion,
		EstimatedCostUSD: o.EstimatedCostUSD, HTTPStatus: o.HTTPStatus,
		LatencyMillis: o.LatencyMillis, RecordedAt: now.UTC(),
	}
	if u := o.Usage; u != nil {
		r.Tokens = &ocrusage.Tokens{Input: u.Input, Output: u.Output, Thinking: u.Thinking,
			CachedInput: u.CachedInput, ToolInput: u.ToolInput, Total: u.Total}
	}
	return r
}
