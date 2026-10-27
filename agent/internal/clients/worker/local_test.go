package worker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker/internal/agentcontract"
	"github.com/stretchr/testify/require"
)

func TestLocalTransportIsUnsignedAndJWTScoped(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("Cookie"))
		require.Equal(t, "request-access", r.Header.Get(agentcontract.AccessTokenHeader))
		fmt.Fprintf(w, `{"actorId":%q,"expiresAt":%q}`, actorID, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	}))
	defer server.Close()
	client, err := NewLocal(server.URL)
	require.NoError(t, err)
	_, err = client.Preflight(WithAccessToken(t.Context(), "request-access"))
	require.NoError(t, err)
	_, err = client.Preflight(t.Context())
	require.Error(t, err)
	require.Equal(t, 1, calls)
	for _, endpoint := range []string{"http://localhost:8000", "http://0.0.0.0:8000", "https://127.0.0.1:8000", "http://192.0.2.1:8000", "http://127.0.0.1:8000?x=y", "http://user:pass@127.0.0.1:8000"} {
		_, err := NewLocal(endpoint)
		require.Error(t, err)
	}
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "synthetic")
	_, err = NewLocal(server.URL)
	require.Error(t, err)
	_, err = client.Preflight(WithAccessToken(t.Context(), "request-access"))
	require.Error(t, err)
	require.Equal(t, 1, calls)
}
