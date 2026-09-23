package exchangerate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestFrankfurterRecommendationsInvertTargetBasedRatesAndCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		require.Equal(t, "CAD", request.URL.Query().Get("base"))
		require.Equal(t, "CAD,TWD", request.URL.Query().Get("quotes"))
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`[
			{"date":"2026-09-21","base":"CAD","quote":"CAD","rate":1},
			{"date":"2026-09-21","base":"CAD","quote":"TWD","rate":23.5}
		]`))
	}))
	defer server.Close()

	client := NewFrankfurterClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()
	client.now = func() time.Time { return time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC) }

	first, err := client.Recommendations(context.Background(), []string{"TWD", "CAD"}, "CAD")
	require.NoError(t, err)
	require.Len(t, first.Recommendations, 2)
	require.True(t, first.Recommendations[0].RecommendedRate.Equal(decimal.NewFromInt(1)))
	require.True(t, first.Recommendations[1].RecommendedRate.Equal(decimal.RequireFromString("0.042553191489362")))
	require.Equal(t, "2026-09-21", first.Recommendations[1].RateSnapshotAt)

	_, err = client.Recommendations(context.Background(), []string{"CAD", "TWD"}, "CAD")
	require.NoError(t, err)
	require.Equal(t, 1, requests)
}
