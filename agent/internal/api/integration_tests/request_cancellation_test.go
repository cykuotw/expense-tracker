package integration_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/agent/internal/api"
	"expense-tracker/agent/internal/fixtures"
	agentruntime "expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
)

func TestClientDisconnectStopsRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	request := localRequest(balanceRequestBody).WithContext(ctx)
	writer := httptest.NewRecorder()
	application, err := agentruntime.New(t.Context(), agentruntime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: fixtures.ModelFactory,
		Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
	})
	if err != nil {
		t.Fatal(err)
	}
	api.NewLocalFixtureHandler(application).ServeHTTP(writer, request)
	if strings.Contains(writer.Body.String(), "RUN_FINISHED") {
		t.Fatal("cancelled run reported success")
	}
	_, _ = io.Copy(io.Discard, writer.Result().Body)
}
