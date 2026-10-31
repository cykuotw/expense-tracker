package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"expense-tracker/agent/internal/models"

	"github.com/stretchr/testify/require"
)

type runOutputApplication struct {
	Application
	cursor string
	calls  int
}

func (app *runOutputApplication) ReadRunOutput(_ context.Context, _, _, after string) ([]models.Output, error) {
	app.calls++
	app.cursor = after
	return []models.Output{}, nil
}

func TestReadRunOutputUsesEventIDCursor(t *testing.T) {
	for _, cursor := range []string{"", uuid.New().String(), "0", "-1", "invalid"} {
		t.Run(cursor, func(t *testing.T) {
			app := &runOutputApplication{}
			handler := New(app)
			request := httptest.NewRequest("GET", "http://127.0.0.1:8090/chat/conversations/chat/runs/run/events?after="+cursor, nil)
			request.SetPathValue("conversation", "chat")
			request.SetPathValue("run", "run")
			response := httptest.NewRecorder()
			handler.HandleReadRunOutput(response, request)
			_, err := uuid.Parse(cursor)
			if cursor == "" || err == nil {
				require.Equal(t, http.StatusOK, response.Code)
				require.Equal(t, cursor, app.cursor)
				require.Equal(t, 1, app.calls)
			} else {
				require.Equal(t, http.StatusBadRequest, response.Code)
				require.Zero(t, app.calls)
			}
		})
	}
}
