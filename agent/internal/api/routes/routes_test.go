package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/agent/internal/api/handlers"
	"expense-tracker/agent/internal/models"

	"github.com/stretchr/testify/require"
)

type conversationApplicationStub struct {
	chatApplication
	operation    string
	conversation string
	run          string
}

func (app *conversationApplicationStub) ListConversations(context.Context) ([]models.Conversation, error) {
	app.operation = "list conversations"
	return []models.Conversation{{ID: "chat"}}, nil
}

func (app *conversationApplicationStub) ListRuns(_ context.Context, conversation string) ([]models.Run, error) {
	app.operation, app.conversation = "list runs", conversation
	return []models.Run{{ID: "run"}}, nil
}

func (app *conversationApplicationStub) ReadRunOutput(_ context.Context, conversation, run, _ string) ([]models.Output, error) {
	app.operation, app.conversation, app.run = "read output", conversation, run
	return []models.Output{{Kind: "text", Delta: "saved answer"}}, nil
}

func (app *conversationApplicationStub) DeleteConversation(_ context.Context, conversation string) error {
	app.operation, app.conversation = "delete conversation", conversation
	return nil
}

func TestRegisteredConversationRoutes(t *testing.T) {
	for _, test := range []struct {
		method, path, operation, conversation, run, body string
	}{
		{http.MethodGet, "/chat/conversations", "list conversations", "", "", `"conversationId":"chat"`},
		{http.MethodGet, "/chat/conversations/chat/runs", "list runs", "chat", "", `"id":"run"`},
		{http.MethodGet, "/chat/conversations/chat/runs/run/events", "read output", "chat", "run", `"delta":"saved answer"`},
		{http.MethodDelete, "/chat/conversations/chat", "delete conversation", "chat", "", `"deleted":true`},
	} {
		t.Run(test.operation, func(t *testing.T) {
			app := &conversationApplicationStub{}
			handler := http.NewServeMux()
			Register(handler, handlers.New(app))
			request := httptest.NewRequest(test.method, "http://127.0.0.1:8090"+test.path, nil)
			request.RemoteAddr = "127.0.0.1:12345"
			denied := httptest.NewRecorder()
			handler.ServeHTTP(denied, request)
			require.Equal(t, http.StatusUnauthorized, denied.Code)
			require.Empty(t, app.operation, "authentication must precede runtime access")

			request.Header.Set("Authorization", "Bearer synthetic-access")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code)
			require.Equal(t, "application/json", response.Header().Get("Content-Type"))
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			require.Contains(t, response.Body.String(), test.body)
			require.Equal(t, test.operation, app.operation)
			require.Equal(t, test.conversation, app.conversation)
			require.Equal(t, test.run, app.run)
		})
	}
}

func TestFixtureRoutesExcludeConversations(t *testing.T) {
	fixture := http.NewServeMux()
	RegisterFixture(fixture, handlers.New(&conversationApplicationStub{}))
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8090/chat/conversations", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()
	fixture.ServeHTTP(response, request)
	require.Equal(t, http.StatusNotFound, response.Code)
}

type chatApplication struct {
	handlers.Application
}

func (chatApplication) Run(context.Context, models.ChatRequest, func(models.Output) error) error {
	return nil
}
