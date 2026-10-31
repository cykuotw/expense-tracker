package expenseapi

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/services/agenttools"
	"expense-tracker/backend/services/auth"
	expensestore "expense-tracker/backend/services/expense/stores"
	groupstore "expense-tracker/backend/services/group/stores"
	"expense-tracker/backend/services/user"
	"expense-tracker/internal/agentcontract"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Opt in only against an isolated, migrated synthetic database. The adapter
// simulates Gateway IAM context; it does not establish cloud IAM enforcement.
func TestPostgresSignedClientWorkerBalance(t *testing.T) {
	if os.Getenv("AGENT_TEST_POSTGRES") != "1" {
		t.Skip("requires isolated migrated PostgreSQL and AGENT_TEST_POSTGRES=1")
	}
	db, err := dbstore.NewPostgreSQLStorage(config.Envs)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(t.Context()))
	owner, other, group, emptyGroup := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	createPostgresFixture(t, db, owner, other, group, emptyGroup)

	router := gin.New()
	agenttools.NewHandler(user.NewStore(db), groupstore.NewStore(db), expensestore.NewStore(db), "arn:aws:iam::123456789012:role/agent").RegisterRoutes(router.Group("/api"))
	client := newClient(t, func(request *http.Request) (*http.Response, error) {
		require.Contains(t, request.Header.Get("Authorization"), "AWS4-HMAC-SHA256")
		headers := make(map[string]string)
		for key, values := range request.Header {
			headers[key] = strings.Join(values, ",")
		}
		route := "GET /api" + agentcontract.PreflightPath
		if strings.Contains(request.URL.Path, "/balance/") {
			route = "GET /api" + agentcontract.BalancePath
		}
		accessor := new(core.RequestAccessorV2)
		trusted, err := accessor.EventToRequestWithContext(request.Context(), events.APIGatewayV2HTTPRequest{
			Version: "2.0", RawPath: request.URL.Path, Headers: headers,
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				RouteKey: route,
				HTTP:     events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: request.Method, Path: request.URL.Path},
				Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{IAM: &events.APIGatewayV2HTTPRequestContextAuthorizerIAMDescription{
					AccountID: "123456789012", UserARN: "arn:aws:sts::123456789012:assumed-role/agent/synthetic",
				}},
			},
		})
		require.NoError(t, err)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, trusted)
		return response.Result(), nil
	})
	ownerToken, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), owner)
	require.NoError(t, err)
	ctx := credentials.WithAccessToken(t.Context(), ownerToken)
	actor, err := client.Preflight(ctx)
	require.NoError(t, err)
	require.Equal(t, owner.String(), actor.ActorID)
	result, err := client.GetBalance(ctx, group.String())
	require.NoError(t, err)
	require.Len(t, result.Balances, 2)
	currencies := map[string]string{}
	for _, balance := range result.Balances {
		currencies[balance.Currency] = balance.Amount
	}
	require.Equal(t, map[string]string{"CAD": "12.34", "USD": "5.67"}, currencies)
	result, err = client.GetBalance(ctx, emptyGroup.String())
	require.NoError(t, err)
	require.Empty(t, result.Balances)
	otherToken, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), other)
	require.NoError(t, err)
	_, err = client.GetBalance(credentials.WithAccessToken(t.Context(), otherToken), emptyGroup.String())
	require.ErrorContains(t, err, "not_found")
	_, err = db.ExecContext(t.Context(), `DELETE FROM group_member WHERE group_id=$1 AND user_id=$2`, group, owner)
	require.NoError(t, err)
	_, err = client.GetBalance(ctx, group.String())
	require.ErrorContains(t, err, "not_found", "each call rechecks current membership")
	_, err = db.ExecContext(t.Context(), `UPDATE users SET is_active=FALSE WHERE id=$1`, owner)
	require.NoError(t, err)
	_, err = client.Preflight(ctx)
	require.ErrorContains(t, err, "forbidden")
	_, err = client.GetBalance(ctx, emptyGroup.String())
	require.ErrorContains(t, err, "forbidden")
}

func createPostgresFixture(t *testing.T, db *sql.DB, owner, other, group, emptyGroup uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		_, err := db.ExecContext(context.Background(), `DELETE FROM groups WHERE id IN ($1,$2)`, group, emptyGroup)
		require.NoError(t, err)
		_, err = db.ExecContext(context.Background(), `DELETE FROM users WHERE id IN ($1,$2)`, owner, other)
		require.NoError(t, err)
	})
	for _, id := range []uuid.UUID{owner, other} {
		name := "agent-test-" + id.String()[:8]
		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, username, firstname, lastname, email, password_hash, create_time_utc, is_active, has_local_password, role)
		VALUES ($1,$2,'Synthetic','Agent',$3,'unused',NOW(),TRUE,TRUE,'user')`, id, name, name+"@example.test")
		require.NoError(t, err)
	}
	for _, id := range []uuid.UUID{group, emptyGroup} {
		_, err := db.ExecContext(t.Context(), `INSERT INTO groups (id, group_name, description, create_time_utc, is_active, create_by_user_id, currency)
		VALUES ($1,'Synthetic agent group','',NOW(),TRUE,$2,'CAD')`, id, owner)
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), `INSERT INTO group_member (id,group_id,user_id) VALUES ($1,$2,$3)`, uuid.New(), id, owner)
		require.NoError(t, err)
	}
	_, err := db.ExecContext(t.Context(), `INSERT INTO group_member (id,group_id,user_id) VALUES ($1,$2,$3)`, uuid.New(), group, other)
	require.NoError(t, err)
	for _, row := range []struct {
		currency, amount string
		outdated         bool
	}{{"CAD", "12.34", false}, {"USD", "5.67", false}, {"CAD", "99.99", true}} {
		_, err := db.ExecContext(t.Context(), `INSERT INTO balance (id,sender_user_id,receiver_user_id,share,group_id,create_time_utc,is_outdated,is_settled,currency)
		VALUES ($1,$2,$3,$4,$5,NOW(),$6,FALSE,$7)`, uuid.New(), owner, other, row.amount, group, row.outdated, row.currency)
		require.NoError(t, err)
	}
}
