package ocrusage

import (
	"context"
	"encoding/json/v2"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stretchr/testify/require"
)

func sampleRecord() Record {
	return Record{RequestID: "1b0685e9-aeea-4033-a2b5-e010c1879657", AccountID: "c0fc455e-8bc8-4446-9b53-55d697c9bc66",
		Model: "gemini-3.8-flash", Outcome: "success", UsageStatus: "available", CostStatus: "estimated",
		PricingVersion: "google-standard-2026-10-20-intro", EstimatedCostUSD: new(0.001),
		Tokens: &Tokens{Input: 100, Output: 20, Thinking: 10, CachedInput: 5, Total: 130}, HTTPStatus: 200, LatencyMillis: 50, RecordedAt: time.Now().UTC()}
}

func TestRecordValidationPreservesUnknownVersusZero(t *testing.T) {
	r := sampleRecord()
	require.NoError(t, r.Validate())
	for _, change := range []func(*Record){
		func(r *Record) { r.RequestID = "not-an-id" }, func(r *Record) { r.AccountID = "" },
		func(r *Record) { r.UsageStatus = "missing" }, func(r *Record) { r.Tokens.Total++ },
		func(r *Record) { r.Tokens.CachedInput = 101 }, func(r *Record) { r.Tokens.Output = -1 },
		func(r *Record) { r.EstimatedCostUSD = new(math.NaN()) }, func(r *Record) { r.EstimatedCostUSD = nil },
		func(r *Record) { r.CostStatus = "pricing_unavailable" }, func(r *Record) { r.PricingVersion = "" },
	} {
		r := sampleRecord()
		change(&r)
		require.Error(t, r.Validate())
	}
	r.Tokens = nil
	r.UsageStatus = "missing"
	r.CostStatus = "usage_unavailable"
	r.PricingVersion = ""
	r.EstimatedCostUSD = nil
	require.NoError(t, r.Validate())
	r = sampleRecord()
	r.Tokens = &Tokens{}
	r.EstimatedCostUSD = new(0.0)
	require.NoError(t, r.Validate(), "actual zero usage remains distinct from unavailable usage")
}

type fakeInvoker struct {
	input  *lambda.InvokeInput
	status int32
	err    error
}

func (f *fakeInvoker) Invoke(_ context.Context, in *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	f.input = in
	return &lambda.InvokeOutput{StatusCode: f.status}, f.err
}

func TestPublisherUsesAsyncIAMEventAndChecksAcceptance(t *testing.T) {
	f := &fakeInvoker{status: 202}
	r := sampleRecord()
	require.NoError(t, Enqueue(t.Context(), f, "worker:live", r))
	require.Equal(t, types.InvocationTypeEvent, f.input.InvocationType)
	require.Equal(t, "worker:live", *f.input.FunctionName)
	var event Event
	require.NoError(t, json.Unmarshal(f.input.Payload, &event))
	require.Equal(t, r.RequestID, event.Usage.RequestID)
	require.Equal(t, EventType, event.Type)
	for _, status := range []int32{0, 200, 500} {
		f.status = status
		require.Error(t, Enqueue(t.Context(), f, "worker:live", r))
	}
	f.err = errors.New("network unavailable")
	require.Error(t, Enqueue(t.Context(), f, "worker:live", r))
	require.Error(t, Enqueue(t.Context(), f, "", r))
}
