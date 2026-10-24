package serverless

import (
	"context"
	"encoding/json/v2"
	"errors"
	"testing"
	"time"

	"expense-tracker/backend/services/ocrusage"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

type fakeUsageStore struct {
	records []ocrusage.Record
	err     error
}

func (f *fakeUsageStore) Save(_ context.Context, r ocrusage.Record) error {
	f.records = append(f.records, r)
	return f.err
}

func TestWorkerUsageEventIsSeparateFromGatewayBody(t *testing.T) {
	r := ocrusage.Record{RequestID: "1b0685e9-aeea-4033-a2b5-e010c1879657", AccountID: "c0fc455e-8bc8-4446-9b53-55d697c9bc66",
		Model: "gemini-3.8-flash", Outcome: "request_failed", UsageStatus: "unavailable", CostStatus: "usage_unavailable", RecordedAt: time.Now().UTC()}
	raw, err := json.Marshal(ocrusage.Event{Type: ocrusage.EventType, Usage: r})
	require.NoError(t, err)
	store := &fakeUsageStore{}
	gatewayCalls := 0
	proxy := func(_ context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		gatewayCalls++
		require.Equal(t, string(raw), req.Body)
		return events.APIGatewayV2HTTPResponse{StatusCode: 401}, nil
	}
	_, err = HandleWorkerEvent(t.Context(), raw, store, proxy)
	require.NoError(t, err)
	require.Len(t, store.records, 1)
	require.Zero(t, gatewayCalls)
	wrapped, err := json.Marshal(events.APIGatewayV2HTTPRequest{Version: "2.0", Body: string(raw)})
	require.NoError(t, err)
	response, err := HandleWorkerEvent(t.Context(), wrapped, store, proxy)
	require.NoError(t, err)
	require.Equal(t, 401, response.(events.APIGatewayV2HTTPResponse).StatusCode)
	require.Len(t, store.records, 1)
	require.Equal(t, 1, gatewayCalls)
	store.err = errors.New("secret database details")
	_, err = HandleWorkerEvent(t.Context(), raw, store, proxy)
	require.EqualError(t, err, "OCR usage persistence failed")
	for _, raw := range []string{`{}`, `{"type":"ocr_usage.v1","usage":{}}`, `{"type":"other"}`, `{"type":"ocr_usage.v1","version":"2.0"}`} {
		_, err := HandleWorkerEvent(t.Context(), []byte(raw), store, proxy)
		require.Error(t, err)
	}
}
