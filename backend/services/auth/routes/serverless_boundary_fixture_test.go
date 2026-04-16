package route

import (
	"context"
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/internal/serverless"
	"expense-tracker/backend/services/middleware"
	"expense-tracker/backend/types"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const (
	boundaryFrontendOrigin = "https://app.example.test"
	boundaryAPIOrigin      = "https://api.example.test"
)

type boundaryFixture struct {
	t            *testing.T
	users        *boundaryUserStore
	refreshStore *refreshStoreState
	adapter      *httpadapter.HandlerAdapterV2
}

type boundaryConfigOptions struct {
	apiPath            string
	googleExchangeMode config.GoogleExchangeMode
}

func withBoundaryConfig(t *testing.T, opts boundaryConfigOptions) {
	t.Helper()

	original := config.Envs
	config.Envs.Mode = "test"
	config.Envs.APIPath = opts.apiPath
	config.Envs.FrontendOrigin = boundaryFrontendOrigin
	config.Envs.CORSAllowedOrigins = []string{boundaryFrontendOrigin}
	config.Envs.CORSAllowCredentials = true
	config.Envs.AuthCookieDomain = ""
	config.Envs.AuthCookieSecure = false
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	config.Envs.GoogleOAuthEnabled = true
	config.Envs.GoogleClientId = "test-google-client-id"
	if opts.googleExchangeMode == "" {
		config.Envs.GoogleExchangeMode = config.GoogleExchangeUpstreamVerified
	} else {
		config.Envs.GoogleExchangeMode = opts.googleExchangeMode
	}

	t.Cleanup(func() {
		config.Envs = original
	})
}

func newBoundaryFixture(t *testing.T, opts boundaryConfigOptions, wrapGoogleAuthorizer bool) *boundaryFixture {
	t.Helper()
	withBoundaryConfig(t, opts)
	gin.SetMode(gin.ReleaseMode)

	users := newBoundaryUserStore(t)
	refreshStore := newRefreshStoreState()
	handler := NewHandler(users, invitationStoreMock(), refreshStore)

	router := gin.New()
	router.Use(middleware.CORSMiddleware())
	public := router.Group(config.Envs.APIPath)
	public.Use(middleware.CSRFMiddleware())
	handler.RegisterRoutes(public)

	var httpHandler http.Handler = router
	if wrapGoogleAuthorizer {
		httpHandler = serverless.WrapWithGoogleAuthorizerClaims(httpHandler)
	}

	return &boundaryFixture{
		t:            t,
		users:        users,
		refreshStore: refreshStore,
		adapter:      httpadapter.NewV2(httpHandler),
	}
}

func (f *boundaryFixture) proxy(req events.APIGatewayV2HTTPRequest) events.APIGatewayV2HTTPResponse {
	f.t.Helper()
	resp, err := f.adapter.ProxyWithContext(context.Background(), req)
	require.NoError(f.t, err)
	return resp
}

func gatewayRequest(method string, path string, opts ...func(*events.APIGatewayV2HTTPRequest)) events.APIGatewayV2HTTPRequest {
	req := events.APIGatewayV2HTTPRequest{
		Version: "2.0",
		RawPath: path,
		Headers: map[string]string{
			"x-forwarded-proto": "https",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			DomainName: strings.TrimPrefix(boundaryAPIOrigin, "https://"),
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method:   method,
				Path:     path,
				SourceIP: "127.0.0.1",
			},
		},
	}
	for _, opt := range opts {
		opt(&req)
	}
	return req
}

func withOrigin(origin string) func(*events.APIGatewayV2HTTPRequest) {
	return withHeader("origin", origin)
}

func withHeader(name string, value string) func(*events.APIGatewayV2HTTPRequest) {
	return func(req *events.APIGatewayV2HTTPRequest) {
		req.Headers[name] = value
	}
}

func withJSONBody(t *testing.T, payload interface{}) func(*events.APIGatewayV2HTTPRequest) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return func(req *events.APIGatewayV2HTTPRequest) {
		req.Headers["content-type"] = "application/json"
		req.Body = string(body)
	}
}

func withCookies(cookies []string) func(*events.APIGatewayV2HTTPRequest) {
	return func(req *events.APIGatewayV2HTTPRequest) {
		req.Cookies = append(req.Cookies, cookies...)
	}
}

func withCookieHeader(cookies []string) func(*events.APIGatewayV2HTTPRequest) {
	return withHeader("cookie", strings.Join(cookies, "; "))
}

func withAuthorizerClaims(claims map[string]string) func(*events.APIGatewayV2HTTPRequest) {
	return func(req *events.APIGatewayV2HTTPRequest) {
		req.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
			JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
				Claims: claims,
			},
		}
	}
}

func responseSetCookies(resp events.APIGatewayV2HTTPResponse) []*http.Cookie {
	values := append([]string{}, resp.Cookies...)
	for name, value := range resp.Headers {
		if strings.EqualFold(name, "set-cookie") && value != "" {
			values = append(values, value)
		}
	}
	for name, valuesFromHeader := range resp.MultiValueHeaders {
		if strings.EqualFold(name, "set-cookie") {
			values = append(values, valuesFromHeader...)
		}
	}

	response := &http.Response{Header: make(http.Header)}
	for _, value := range values {
		response.Header.Add("Set-Cookie", value)
	}
	return response.Cookies()
}

func cookieValue(t *testing.T, cookies []*http.Cookie, name string) string {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	t.Fatalf("missing cookie %q in %#v", name, cookies)
	return ""
}

func requireCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("missing cookie %q in %#v", name, cookies)
	return nil
}

func replayCookies(cookies []*http.Cookie, names ...string) []string {
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}

	result := make([]string, 0, len(names))
	for _, cookie := range cookies {
		if wanted[cookie.Name] && cookie.Value != "" && cookie.MaxAge >= 0 {
			result = append(result, cookie.Name+"="+cookie.Value)
		}
	}
	return result
}

func issueCSRF(t *testing.T, fixture *boundaryFixture, path string) (string, []string, events.APIGatewayV2HTTPResponse) {
	t.Helper()
	resp := fixture.proxy(gatewayRequest(http.MethodGet, path, withOrigin(boundaryFrontendOrigin)))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookies := responseSetCookies(resp)
	csrf := cookieValue(t, cookies, middleware.CSRFCookieName)
	return csrf, replayCookies(cookies, middleware.CSRFCookieName), resp
}

func loginThroughBoundary(t *testing.T, fixture *boundaryFixture, path string, csrf string, csrfCookies []string) ([]string, events.APIGatewayV2HTTPResponse) {
	t.Helper()
	resp := fixture.proxy(gatewayRequest(http.MethodPost, path,
		withOrigin(boundaryFrontendOrigin),
		withCookies(csrfCookies),
		withHeader(middleware.CSRFHeaderName, csrf),
		withJSONBody(t, types.LoginUserPayload{Email: "user@example.test", Password: "testpassword"}),
	))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookies := responseSetCookies(resp)
	requireCookie(t, cookies, "access_token")
	requireCookie(t, cookies, "refresh_token")
	return replayCookies(cookies, "access_token", "refresh_token"), resp
}

func cookieNames(cookies []*http.Cookie, names ...string) []string {
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}
	var result []string
	for _, cookie := range cookies {
		if wanted[cookie.Name] {
			result = append(result, cookie.Name)
		}
	}
	return result
}
