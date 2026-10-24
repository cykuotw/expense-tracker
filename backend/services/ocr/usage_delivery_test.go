package ocr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUsageRecordRetainsCapabilityIdentityAndUnavailableCounts(t *testing.T) {
	ctx := withUsageIdentity(t.Context(), "c0fc455e-8bc8-4446-9b53-55d697c9bc66", "1b0685e9-aeea-4033-a2b5-e010c1879657")
	o := GeminiUsageObservation{Model: DefaultGeminiModel, Outcome: "request_failed", UsageStatus: "unavailable"}
	o.estimateCost(time.Now())
	r := UsageRecord(ctx, o, time.Now())
	require.NoError(t, r.Validate())
	require.Nil(t, r.Tokens)
	require.Nil(t, r.EstimatedCostUSD)
	require.Equal(t, "c0fc455e-8bc8-4446-9b53-55d697c9bc66", r.AccountID)
	o = GeminiUsageObservation{Model: DefaultGeminiModel, Outcome: "success", UsageStatus: "available",
		Usage: &GeminiTokenUsage{Input: 100, Output: 20, Thinking: 10, Total: 130, standardTier: true}}
	o.estimateCost(time.Now())
	r = UsageRecord(ctx, o, time.Now())
	require.NoError(t, r.Validate())
	require.Equal(t, int64(10), r.Tokens.Thinking)
	require.NotNil(t, r.EstimatedCostUSD)
	require.NotEmpty(t, r.PricingVersion)
	require.Error(t, UsageRecord(t.Context(), o, time.Now()).Validate(), "no unverified identity can be published")
}

type identityProvider struct{ identity usageIdentity }

func (p *identityProvider) AnalyzeExpense(ctx context.Context, _ []byte) (Result, error) {
	p.identity, _ = ctx.Value(usageIdentityKey{}).(usageIdentity)
	return Result{}, ErrProviderRequestFailed
}

func TestDraftBindsUsageToVerifiedCapabilityEvenWhenProviderFails(t *testing.T) {
	now, account, request, token, secret := draftFixture(t, "image/jpeg")
	provider := &identityProvider{}
	h := NewDraftHandler(RuntimeConfig{FrontendOrigin: "https://app.example.com", CapabilitySecret: secret}, &replayStub{}, provider, nil)
	h.now = func() time.Time { return now.Add(time.Second) }
	response, err := h.Handle(t.Context(), stubRequest(token, account, request, "image/jpeg", encodeTestJPEG(t, 16, 12)))
	require.NoError(t, err)
	require.Equal(t, 503, response.StatusCode)
	require.Equal(t, usageIdentity{account, request}, provider.identity)
}
