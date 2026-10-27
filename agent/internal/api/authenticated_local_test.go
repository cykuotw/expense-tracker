package api

import (
	"context"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/agent/internal/clients/worker"
	"expense-tracker/agent/internal/models"
	agentruntime "expense-tracker/agent/internal/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
)

type observedLocalModel struct {
	model.LLM
	calls *int
}

func (m observedLocalModel) GenerateContent(ctx context.Context, request *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	*m.calls++
	return m.LLM.GenerateContent(ctx, request, stream)
}

func TestLocalChatRequiresCredentialsAndLocalBoundary(t *testing.T) {
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "invalid-access", r.Header.Get("X-Agent-Access-Token"))
		w.WriteHeader(401)
	}))
	defer backend.Close()
	client, err := worker.NewLocal(backend.URL)
	require.NoError(t, err)
	modelCalls := 0
	app, err := agentruntime.New(t.Context(), agentruntime.Config{
		Balance: client, Models: func(ctx context.Context, skill, name string) (model.LLM, error) {
			base, err := models.FixtureFactory(ctx, skill, name)
			return observedLocalModel{LLM: base, calls: &modelCalls}, err
		},
	})
	require.NoError(t, err)
	handler, err := NewLocal(app)
	require.NoError(t, err)
	for _, tc := range []struct {
		name, authorization, peer, host, extraHeader string
		status                                       int
	}{
		{"missing", "", "127.0.0.1:9000", "127.0.0.1:8090", "", 401},
		{"bad scheme", "Basic abc", "127.0.0.1:9000", "127.0.0.1:8090", "", 401},
		{"remote peer", "Bearer invalid-access", "192.0.2.1:9000", "127.0.0.1:8090", "", 403},
		{"foreign host", "Bearer invalid-access", "127.0.0.1:9000", "remote.test:8090", "", 403},
		{"forged actor", "Bearer invalid-access", "127.0.0.1:9000", "127.0.0.1:8090", "X-Actor-ID", 400},
		{"cookie", "Bearer invalid-access", "127.0.0.1:9000", "127.0.0.1:8090", "Cookie", 400},
		{"invalid JWT preflight", "Bearer invalid-access", "127.0.0.1:9000", "127.0.0.1:8090", "", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "http://"+tc.host+"/chat", strings.NewReader(`{"conversationId":"local","messageId":"one","text":"balance demo"}`))
			request.RemoteAddr = tc.peer
			request.Header.Set("Content-Type", "application/json")
			if tc.authorization != "" {
				request.Header.Set("Authorization", tc.authorization)
			}
			if tc.extraHeader != "" {
				request.Header.Set(tc.extraHeader, "forged")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code)
			require.NotContains(t, response.Body.String(), "RUN_STARTED")
			require.NotContains(t, response.Body.String(), "invalid-access")
		})
	}
	require.Equal(t, 1, calls)
	require.Zero(t, modelCalls, "preflight rejects runs before inference")
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "synthetic")
	_, err = NewLocal(app)
	require.Error(t, err)
}
