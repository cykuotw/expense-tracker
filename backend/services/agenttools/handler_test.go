package agenttools

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/internal/agentcontract"
	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const testRole = "arn:aws:iam::123456789012:role/agent"
const testPrincipal = "arn:aws:sts::123456789012:assumed-role/agent/synthetic"

var testActor = uuid.MustParse("11111111-1111-4111-8111-111111111111")
var testGroup = uuid.MustParse("22222222-2222-4222-8222-222222222222")

type fixture struct {
	active, member bool
	rows           []types.Balance
	reads          int
	storeErr       error
}

func (f *fixture) GetUserByID(id string) (*types.User, error) {
	if id != testActor.String() {
		return nil, errors.New("unknown actor")
	}
	return &types.User{IsActive: f.active}, f.storeErr
}
func (f *fixture) CheckGroupUserPairExist(group, actor string) (bool, error) {
	return f.member && group == testGroup.String() && actor == testActor.String(), f.storeErr
}
func (f *fixture) GetBalanceByGroupId(string) ([]types.Balance, error) {
	f.reads++
	return f.rows, f.storeErr
}

func token(t *testing.T, actor, kind, secret string, expiry *jwt.NumericDate) string {
	t.Helper()
	claims := auth.Claims{UserID: actor, TokenType: kind, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: expiry}}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return value
}

// Only this test adapter fabricates Gateway context. No application HTTP header
// can populate the Lambda context, and this does not verify AWS IAM signatures.
func gatewayRequest(t *testing.T, method, path, route, principal, access string) *http.Request {
	t.Helper()
	accessor := new(core.RequestAccessorV2)
	request, err := accessor.EventToRequestWithContext(context.Background(), events.APIGatewayV2HTTPRequest{
		Version: "2.0", RawPath: path, Headers: map[string]string{agentcontract.AccessTokenHeader: access},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RouteKey: route, HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: method, Path: path},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{IAM: &events.APIGatewayV2HTTPRequestContextAuthorizerIAMDescription{
				AccountID: "123456789012", UserARN: principal,
			}},
		},
	})
	require.NoError(t, err)
	return request
}

func fixtureRouter(f *fixture, role string) *gin.Engine {
	router := gin.New()
	NewHandler(f, f, f, role).RegisterRoutes(router.Group(""))
	return router
}

func TestAuthorizationBoundary(t *testing.T) {
	valid := token(t, testActor.String(), "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(time.Hour)))
	expired := token(t, testActor.String(), "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(-time.Hour)))
	for _, tc := range []struct {
		name, principal, route, token string
		inactive                      bool
		status                        int
	}{
		{"approved", testPrincipal, "GET " + agentcontract.PreflightPath, valid, false, 200},
		{"missing principal", "", "GET " + agentcontract.PreflightPath, valid, false, 403},
		{"wrong role", strings.Replace(testPrincipal, "/agent/", "/other/", 1), "GET " + agentcontract.PreflightPath, valid, false, 403},
		{"wrong account", strings.Replace(testPrincipal, "123456789012", "999999999999", 1), "GET " + agentcontract.PreflightPath, valid, false, 403},
		{"role prefix", strings.Replace(testPrincipal, "/agent/", "/agent-evil/", 1), "GET " + agentcontract.PreflightPath, valid, false, 403},
		{"default route", testPrincipal, "$default", valid, false, 403},
		{"greedy route", testPrincipal, "ANY /{proxy+}", valid, false, 403},
		{"wrong method context", testPrincipal, "POST " + agentcontract.PreflightPath, valid, false, 403},
		{"missing JWT", testPrincipal, "GET " + agentcontract.PreflightPath, "", false, 401},
		{"malformed JWT", testPrincipal, "GET " + agentcontract.PreflightPath, "broken", false, 401},
		{"expired JWT", testPrincipal, "GET " + agentcontract.PreflightPath, expired, false, 401},
		{"forged actor", testPrincipal, "GET " + agentcontract.PreflightPath, token(t, uuid.NewString(), "access", "forged", jwt.NewNumericDate(time.Now().Add(time.Hour))), false, 401},
		{"refresh JWT", testPrincipal, "GET " + agentcontract.PreflightPath, token(t, testActor.String(), "refresh", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(time.Hour))), false, 401},
		{"no expiry", testPrincipal, "GET " + agentcontract.PreflightPath, token(t, testActor.String(), "access", config.Envs.JWTSecret, nil), false, 401},
		{"invalid actor", testPrincipal, "GET " + agentcontract.PreflightPath, token(t, "arbitrary", "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(time.Hour))), false, 401},
		{"inactive account", testPrincipal, "GET " + agentcontract.PreflightPath, valid, true, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fixture{active: !tc.inactive, member: true}
			request := gatewayRequest(t, "GET", agentcontract.PreflightPath, tc.route, tc.principal, tc.token)
			request.Header.Set("X-Actor-ID", uuid.NewString())
			request.Header.Set("X-Agent-Principal", testPrincipal)
			response := httptest.NewRecorder()
			fixtureRouter(f, testRole).ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			require.NotContains(t, response.Body.String(), valid)
			if response.Code == 200 {
				var actor agentcontract.Preflight
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &actor))
				require.Equal(t, testActor.String(), actor.ActorID)
				require.True(t, actor.ExpiresAt.After(time.Now()))
			}
		})
	}

	for _, role := range []string{testRole, "", "not-a-role"} {
		request := httptest.NewRequest("GET", agentcontract.PreflightPath, nil)
		request.Header.Set(agentcontract.AccessTokenHeader, valid)
		request.Header.Set("X-Amzn-Request-Context", `{"iam":{"userArn":"`+testPrincipal+`"}}`)
		response := httptest.NewRecorder()
		fixtureRouter(&fixture{active: true}, role).ServeHTTP(response, request)
		require.Equal(t, 403, response.Code)
	}
}

func TestBalanceAuthorizationCurrencyAndBounds(t *testing.T) {
	valid := token(t, testActor.String(), "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(time.Hour)))
	f := &fixture{active: true, member: true}
	for _, currency := range []string{"CAD", "USD"} {
		f.rows = append(f.rows, types.Balance{GroupID: testGroup, SenderUserID: testActor, ReceiverUserID: uuid.New(), Share: decimal.RequireFromString("12.34"), Currency: currency})
	}
	router := fixtureRouter(f, testRole)
	call := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, gatewayRequest(t, "GET", path, "GET "+agentcontract.BalancePath, testPrincipal, valid))
		return response
	}
	path := agentcontract.Prefix + "/balance/" + testGroup.String()
	response := call(path)
	require.Equal(t, 200, response.Code)
	var result agentcontract.BalanceResult
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Equal(t, []agentcontract.Balance{
		{From: testActor.String(), To: f.rows[0].ReceiverUserID.String(), Amount: "12.34", Currency: "CAD"},
		{From: testActor.String(), To: f.rows[1].ReceiverUserID.String(), Amount: "12.34", Currency: "USD"},
	}, result.Balances)
	f.member = false
	require.Equal(t, 404, call(path).Code)
	require.Equal(t, 1, f.reads)
	f.member = true
	f.active = false
	require.Equal(t, 403, call(path).Code)
	require.Equal(t, 1, f.reads)
	f.active = true
	require.Equal(t, 400, call(agentcontract.Prefix+"/balance/invalid").Code)
	f.rows = nil
	response = call(path)
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `"balances":[]`)
	f.rows = make([]types.Balance, agentcontract.MaxBalanceRows+1)
	require.Equal(t, 422, call(path).Code)
	f.rows = []types.Balance{{GroupID: testGroup, Currency: "CAD", Share: decimal.NewFromInt(-1)}}
	require.Equal(t, 500, call(path).Code)
	f.storeErr = errors.New("private database detail")
	response = call(path)
	require.NotContains(t, response.Body.String(), "private database detail")
}

func TestForgedPayloadCookieAndAlternateMethods(t *testing.T) {
	valid := token(t, testActor.String(), "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(time.Hour)))
	router := fixtureRouter(&fixture{active: true}, testRole)
	for _, method := range []string{"POST", "PUT", "DELETE", "HEAD"} {
		request := gatewayRequest(t, method, agentcontract.PreflightPath, "GET "+agentcontract.PreflightPath, testPrincipal, valid)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.NotEqual(t, 200, response.Code)
	}
	request := gatewayRequest(t, "GET", agentcontract.PreflightPath, "GET "+agentcontract.PreflightPath, testPrincipal, "")
	request.Header.Set("Authorization", "Bearer "+valid)
	request.AddCookie(&http.Cookie{Name: "access_token", Value: valid})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 401, response.Code)
	request = gatewayRequest(t, "GET", agentcontract.PreflightPath, "GET "+agentcontract.PreflightPath, testPrincipal, valid)
	request.URL.RawQuery = "actorId=forged"
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 400, response.Code)
	request.URL.RawQuery = ""
	request.Header.Add(agentcontract.AccessTokenHeader, valid)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 401, response.Code)
}

func TestTrustedContextMissingIAMAndDisabledRole(t *testing.T) {
	valid := token(t, testActor.String(), "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(time.Hour)))
	for _, authorizer := range []*events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		nil, {}, {JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"principal": testPrincipal}}},
	} {
		request, err := new(core.RequestAccessorV2).EventToRequestWithContext(t.Context(), events.APIGatewayV2HTTPRequest{
			Version: "2.0", RawPath: agentcontract.PreflightPath, Headers: map[string]string{agentcontract.AccessTokenHeader: valid},
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				RouteKey: "GET " + agentcontract.PreflightPath, Authorizer: authorizer,
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "GET", Path: agentcontract.PreflightPath},
			},
		})
		require.NoError(t, err)
		response := httptest.NewRecorder()
		fixtureRouter(&fixture{active: true}, testRole).ServeHTTP(response, request)
		require.Equal(t, 403, response.Code)
	}
	for _, role := range []string{"", "malformed-role", "arn:aws:iam::123456789012:role/wrong"} {
		request := gatewayRequest(t, "GET", agentcontract.PreflightPath, "GET "+agentcontract.PreflightPath, testPrincipal, valid)
		response := httptest.NewRecorder()
		fixtureRouter(&fixture{active: true}, role).ServeHTTP(response, request)
		require.Equal(t, 403, response.Code)
	}
	request := gatewayRequest(t, "POST", agentcontract.PreflightPath, "GET "+agentcontract.PreflightPath, testPrincipal, valid)
	request.Method = "GET"
	response := httptest.NewRecorder()
	fixtureRouter(&fixture{active: true}, testRole).ServeHTTP(response, request)
	require.Equal(t, 403, response.Code)
	request = gatewayRequest(t, "GET", agentcontract.PreflightPath, "GET "+agentcontract.PreflightPath, testPrincipal, valid)
	request.Body = io.NopCloser(strings.NewReader(`{"actorId":"forged"}`))
	request.ContentLength = 20
	response = httptest.NewRecorder()
	fixtureRouter(&fixture{active: true}, testRole).ServeHTTP(response, request)
	require.Equal(t, 400, response.Code)
}
