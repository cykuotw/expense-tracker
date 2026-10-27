package tracker

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/internal/agentcontract"
	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/stretchr/testify/require"
)

func TestAgentToolsWiringPreservesOrdinaryBalance(t *testing.T) {
	db := openAuthorizationTestDB(t)
	fixture := createAuthorizationFixture(t, db)
	t.Setenv("AGENT_TOOL_ROLE_ARN", "arn:aws:iam::123456789012:role/agent")
	var logs bytes.Buffer
	handler := NewHandler(db, WithLogger(slog.New(slog.NewJSONHandler(&logs, nil))))
	ordinary := serveAuthenticatedRequest(t, handler, "GET", config.Envs.APIPath+"/balance/"+fixture.bobGroup.String(), fixture.bobID)
	require.Equal(t, http.StatusOK, ordinary.Code, ordinary.Body.String())
	require.Contains(t, ordinary.Body.String(), `"CAD"`)
	access, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), fixture.bobID)
	require.NoError(t, err)
	path := config.Envs.APIPath + agentcontract.Prefix + "/balance/" + fixture.bobGroup.String()
	request := httptest.NewRequest("GET", path, nil)
	request.Header.Set(agentcontract.AccessTokenHeader, access)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, 403, response.Code, "JWT alone never authorizes internal tools")
	trusted, err := new(core.RequestAccessorV2).EventToRequestWithContext(t.Context(), events.APIGatewayV2HTTPRequest{
		Version: "2.0", RawPath: path, Headers: map[string]string{agentcontract.AccessTokenHeader: access},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RouteKey: "GET " + config.Envs.APIPath + agentcontract.BalancePath,
			HTTP:     events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "GET", Path: path},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{IAM: &events.APIGatewayV2HTTPRequestContextAuthorizerIAMDescription{
				AccountID: "123456789012", UserARN: "arn:aws:sts::123456789012:assumed-role/agent/synthetic",
			}},
		},
	})
	require.NoError(t, err)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, trusted)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"amount":"10"`)
	require.Contains(t, response.Body.String(), `"currency":"CAD"`)
	require.NotContains(t, logs.String(), access)
	require.NotContains(t, logs.String(), "X-Agent-Access-Token")
}
