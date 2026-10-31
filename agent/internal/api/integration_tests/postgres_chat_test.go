package integration_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"expense-tracker/agent/internal/adapters/expenseapi"
	"expense-tracker/agent/internal/api"
	"expense-tracker/agent/internal/databasebootstrap"
	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	agentruntime "expense-tracker/agent/internal/runtime"
	sessionstore "expense-tracker/agent/internal/store/session"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/services/agenttools"
	"expense-tracker/backend/services/auth"
	expensestore "expense-tracker/backend/services/expense/stores"
	groupstore "expense-tracker/backend/services/group/stores"
	"expense-tracker/backend/services/user"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestLocalChatPostgresRoundTrip(t *testing.T) {
	if os.Getenv("AGENT_TEST_POSTGRES") != "1" {
		t.Skip("requires isolated migrated PostgreSQL and AGENT_TEST_POSTGRES=1")
	}
	db, err := dbstore.NewPostgreSQLStorage(config.Envs)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(t.Context()))
	owner, other, group := uuid.New(), uuid.New(), uuid.New()
	t.Cleanup(func() {
		_, err := db.ExecContext(context.Background(), `DELETE FROM groups WHERE id=$1`, group.String())
		require.NoError(t, err)
		_, err = db.ExecContext(context.Background(), `DELETE FROM users WHERE id IN ($1,$2)`, owner.String(), other.String())
		require.NoError(t, err)
	})
	for _, id := range []uuid.UUID{owner, other} {
		name := "local-agent-" + id.String()[:8]
		_, err = db.ExecContext(t.Context(), `INSERT INTO users (id,username,firstname,lastname,email,password_hash,create_time_utc,is_active,has_local_password,role)
		VALUES ($1,$2,'Local','Synthetic',$3,'unused',NOW(),TRUE,TRUE,'user')`, id.String(), name, name+"@example.test")
		require.NoError(t, err)
	}
	_, err = db.ExecContext(t.Context(), `INSERT INTO groups (id,group_name,description,create_time_utc,is_active,create_by_user_id,currency)
	VALUES ($1,'Local synthetic','',NOW(),TRUE,$2,'CAD')`, group.String(), owner.String())
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `INSERT INTO group_member (id,group_id,user_id) VALUES ($1,$2,$3)`, uuid.New().String(), group.String(), owner.String())
	require.NoError(t, err)
	for _, currency := range []string{"CAD", "USD"} {
		_, err = db.ExecContext(t.Context(), `INSERT INTO balance (id,sender_user_id,receiver_user_id,share,group_id,create_time_utc,is_outdated,is_settled,currency)
		VALUES ($1,$2,$3,12.34,$4,NOW(),FALSE,FALSE,$5)`, uuid.New().String(), owner.String(), other.String(), group.String(), currency)
		require.NoError(t, err)
	}
	handler, err := agenttools.NewLocalHandler(user.NewStore(db), groupstore.NewStore(db), expensestore.NewStore(db))
	require.NoError(t, err)
	router := gin.New()
	handler.RegisterRoutes(router.Group(""))
	backend := httptest.NewServer(router)
	defer backend.Close()
	client, err := expenseapi.NewLocal(backend.URL)
	require.NoError(t, err)
	var logs bytes.Buffer
	deps := agentruntime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Tools: testCatalog(t, client), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Authenticate: client.Authenticate, Models: fixtures.ModelFactory,
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
	}
	if os.Getenv("AGENT_STATE_DATABASE_URL") != "" {
		require.NoError(t, databasebootstrap.ProvisionRole(t.Context(), db, "agent_state_test", "synthetic_state_password"))
		state, err := sessionstore.Open(t.Context(), os.Getenv("AGENT_STATE_DATABASE_URL"))
		require.NoError(t, err)
		defer state.Close()
		deps.Sessions = state
		deps.Records = state
	}
	app, err := agentruntime.New(t.Context(), deps)
	require.NoError(t, err)
	chatHandler, err := api.NewLocalHandler(app)
	require.NoError(t, err)
	chat := httptest.NewServer(chatHandler)
	defer chat.Close()
	makeToken := func(actor uuid.UUID, expiry time.Time) string {
		claims := auth.Claims{UserID: actor.String(), TokenType: "access", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiry)}}
		value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.Envs.JWTSecret))
		require.NoError(t, err)
		return value
	}
	access := makeToken(owner, time.Now().Add(time.Hour))
	call := func(token, message string) (int, string) {
		request, err := http.NewRequestWithContext(t.Context(), "POST", chat.URL+"/chat", strings.NewReader(`{"conversationId":"local","messageId":"`+message+`","text":"balance `+group.String()+`"}`))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := chat.Client().Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NotContains(t, string(body), access)
		return response.StatusCode, string(body)
	}
	status, body := call(access, "success")
	require.Equal(t, 200, status, body)
	require.Contains(t, body, "RUN_FINISHED")
	for _, value := range []string{"12.34", "CAD", "USD"} {
		require.Contains(t, body, value)
	}
	require.NotContains(t, logs.String(), access)
	for _, invalid := range []string{"", "invalid-access", makeToken(owner, time.Now().Add(-time.Hour))} {
		status, body = call(invalid, "invalid")
		require.Equal(t, 401, status, body)
		require.NotContains(t, body, "RUN_STARTED")
	}
	status, body = call(makeToken(other, time.Now().Add(time.Hour)), "outsider")
	require.Equal(t, 200, status)
	require.Contains(t, body, "RUN_ERROR")
	require.NotContains(t, body, "12.34")
	if deps.Records != nil {
		requestHistory := func(method, path, token string) (int, []byte) {
			r, err := http.NewRequestWithContext(t.Context(), method, chat.URL+path, nil)
			require.NoError(t, err)
			r.Header.Set("Authorization", "Bearer "+token)
			response, err := chat.Client().Do(r)
			require.NoError(t, err)
			defer response.Body.Close()
			data, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			return response.StatusCode, data
		}
		status, data := requestHistory("GET", "/chat/conversations/local/runs", access)
		require.Equal(t, 200, status, string(data))
		var runs []models.Run
		require.NoError(t, json.Unmarshal(data, &runs))
		require.Len(t, runs, 1)
		status, data = requestHistory("GET", "/chat/conversations/local/runs/"+runs[0].ID+"/events", access)
		require.Equal(t, 200, status, string(data))
		require.Contains(t, string(data), `"kind":"text"`)
		require.NotContains(t, string(data), "model.request")
		require.NotContains(t, string(data), access)
		status, _ = requestHistory("GET", "/chat/conversations/local/runs/"+runs[0].ID+"/events", makeToken(other, time.Now().Add(time.Hour)))
		require.Equal(t, 404, status)
		status, _ = requestHistory("DELETE", "/chat/conversations/local", access)
		require.Equal(t, 200, status)
		status, _ = requestHistory("GET", "/chat/conversations/local/runs", access)
		require.Equal(t, 404, status)
	}
	_, err = db.ExecContext(t.Context(), `UPDATE users SET is_active=FALSE WHERE id=$1`, owner.String())
	require.NoError(t, err)
	status, body = call(access, "inactive")
	require.Equal(t, 403, status, body)
	require.NotContains(t, body, "RUN_STARTED")
}
