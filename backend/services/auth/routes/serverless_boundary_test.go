package route

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/middleware"
	"expense-tracker/backend/types"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The whole browser auth lifecycle works through the API Gateway v2 adapter.
func TestBoundaryFullAuthFlow(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: ""}, false)

	// Issue the CSRF token and verify the cookie shape the browser will store.
	csrf, csrfCookies, csrfResp := issueCSRF(t, fixture, "/auth/csrf")
	csrfCookie := requireCookie(t, responseSetCookies(csrfResp), middleware.CSRFCookieName)
	assert.Equal(t, "/", csrfCookie.Path)
	assert.True(t, csrfCookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, csrfCookie.SameSite)

	// Log in with the CSRF pair and verify both auth cookies cross the Lambda boundary.
	sessionCookies, loginResp := loginThroughBoundary(t, fixture, "/login", csrf, csrfCookies)
	loginCookies := responseSetCookies(loginResp)
	assert.True(t, requireCookie(t, loginCookies, "access_token").Secure)
	assert.True(t, requireCookie(t, loginCookies, "refresh_token").Secure)

	// Replay the session cookies into a protected request.
	meResp := fixture.proxy(gatewayRequest(http.MethodGet, "/auth/me",
		withOrigin(boundaryFrontendOrigin),
		withCookies(sessionCookies),
	))
	require.Equal(t, http.StatusOK, meResp.StatusCode)

	// Refresh must accept the refresh cookie only when the CSRF pair is present.
	refreshResp := fixture.proxy(gatewayRequest(http.MethodPost, "/auth/refresh",
		withOrigin(boundaryFrontendOrigin),
		withCookies(append(sessionCookies, csrfCookies...)),
		withHeader(middleware.CSRFHeaderName, csrf),
	))
	require.Equal(t, http.StatusOK, refreshResp.StatusCode)
	refreshCookies := responseSetCookies(refreshResp)
	requireCookie(t, refreshCookies, "access_token")
	requireCookie(t, refreshCookies, "refresh_token")

	// Logout must clear the same scoped cookies that login set.
	logoutResp := fixture.proxy(gatewayRequest(http.MethodPost, "/logout",
		withOrigin(boundaryFrontendOrigin),
		withCookies(append(sessionCookies, csrfCookies...)),
		withHeader(middleware.CSRFHeaderName, csrf),
	))
	require.Equal(t, http.StatusOK, logoutResp.StatusCode)
	logoutCookies := responseSetCookies(logoutResp)
	assert.Equal(t, -1, requireCookie(t, logoutCookies, "access_token").MaxAge)
	assert.Equal(t, -1, requireCookie(t, logoutCookies, "refresh_token").MaxAge)

	// After logout, a protected request without valid cookies must fail.
	afterLogoutResp := fixture.proxy(gatewayRequest(http.MethodGet, "/auth/me",
		withOrigin(boundaryFrontendOrigin),
	))
	assert.Equal(t, http.StatusUnauthorized, afterLogoutResp.StatusCode)
}

// State-changing auth routes still require matching CSRF header/cookie pairs.
func TestBoundaryCSRF(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: ""}, false)
	csrf, csrfCookies, _ := issueCSRF(t, fixture, "/auth/csrf")

	loginPayload := types.LoginUserPayload{Email: "user@example.test", Password: "testpassword"}

	// Missing CSRF should stop login before credentials matter.
	missingResp := fixture.proxy(gatewayRequest(http.MethodPost, "/login",
		withOrigin(boundaryFrontendOrigin),
		withJSONBody(t, loginPayload),
	))
	assert.Equal(t, http.StatusForbidden, missingResp.StatusCode)

	// A header/cookie mismatch should be rejected as invalid CSRF.
	mismatchResp := fixture.proxy(gatewayRequest(http.MethodPost, "/login",
		withOrigin(boundaryFrontendOrigin),
		withCookies(csrfCookies),
		withHeader(middleware.CSRFHeaderName, "different-token"),
		withJSONBody(t, loginPayload),
	))
	assert.Equal(t, http.StatusForbidden, mismatchResp.StatusCode)

	sessionCookies, _ := loginThroughBoundary(t, fixture, "/login", csrf, csrfCookies)

	// Refresh is under /auth but is explicitly not an origin-only CSRF exception.
	refreshMissingCSRF := fixture.proxy(gatewayRequest(http.MethodPost, "/auth/refresh",
		withOrigin(boundaryFrontendOrigin),
		withCookies(sessionCookies),
	))
	assert.Equal(t, http.StatusForbidden, refreshMissingCSRF.StatusCode)

	// Logout is also state-changing and must require CSRF.
	logoutMissingCSRF := fixture.proxy(gatewayRequest(http.MethodPost, "/logout",
		withOrigin(boundaryFrontendOrigin),
		withCookies(sessionCookies),
	))
	assert.Equal(t, http.StatusForbidden, logoutMissingCSRF.StatusCode)
}

// The Lambda authorizer wrapper supplies Google claims to the exchange route.
func TestBoundaryGoogleExchangeUpstreamClaims(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: "", googleExchangeMode: config.GoogleExchangeUpstreamVerified}, true)

	// Simulate API Gateway authorizer claims instead of making a live Google call.
	resp := fixture.proxy(gatewayRequest(http.MethodPost, "/auth/google/exchange",
		withOrigin(boundaryFrontendOrigin),
		withAuthorizerClaims(map[string]string{
			"sub":            "google-sub-123",
			"email":          "google-user@example.test",
			"email_verified": "true",
			"given_name":     "Google",
			"family_name":    "User",
		}),
	))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookies := responseSetCookies(resp)
	requireCookie(t, cookies, "access_token")
	requireCookie(t, cookies, "refresh_token")

	// The app session created by Google exchange should behave like a normal login.
	sessionCookies := replayCookies(cookies, "access_token", "refresh_token")
	meResp := fixture.proxy(gatewayRequest(http.MethodGet, "/auth/me",
		withOrigin(boundaryFrontendOrigin),
		withCookies(sessionCookies),
	))
	assert.Equal(t, http.StatusOK, meResp.StatusCode)
}

// Credentialed CORS headers survive the API Gateway v2 adapter.
func TestBoundaryCORS(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: ""}, false)

	// Allowed preflight requests should get credentialed CORS headers.
	allowedPreflight := fixture.proxy(gatewayRequest(http.MethodOptions, "/login",
		withOrigin(boundaryFrontendOrigin),
		withHeader("access-control-request-method", http.MethodPost),
	))
	assert.Equal(t, http.StatusNoContent, allowedPreflight.StatusCode)
	assert.Equal(t, boundaryFrontendOrigin, allowedPreflight.Headers["Access-Control-Allow-Origin"])
	assert.Equal(t, "true", allowedPreflight.Headers["Access-Control-Allow-Credentials"])

	// Disallowed preflight requests should be blocked before handlers run.
	disallowedPreflight := fixture.proxy(gatewayRequest(http.MethodOptions, "/login",
		withOrigin("https://evil.example.test"),
		withHeader("access-control-request-method", http.MethodPost),
	))
	assert.Equal(t, http.StatusForbidden, disallowedPreflight.StatusCode)

	// Allowed actual requests should include credentialed CORS headers.
	allowedActual := fixture.proxy(gatewayRequest(http.MethodGet, "/auth/csrf", withOrigin(boundaryFrontendOrigin)))
	assert.Equal(t, http.StatusOK, allowedActual.StatusCode)
	assert.Equal(t, boundaryFrontendOrigin, allowedActual.Headers["Access-Control-Allow-Origin"])
	assert.Equal(t, "true", allowedActual.Headers["Access-Control-Allow-Credentials"])

	// Disallowed actual requests may route, but must not include allow-credential headers.
	disallowedActual := fixture.proxy(gatewayRequest(http.MethodGet, "/auth/csrf", withOrigin("https://evil.example.test")))
	assert.Equal(t, http.StatusOK, disallowedActual.StatusCode)
	assert.Empty(t, disallowedActual.Headers["Access-Control-Allow-Origin"])
	assert.Empty(t, disallowedActual.Headers["Access-Control-Allow-Credentials"])
}

// APIPath prefixes do not break route matching through the adapter.
func TestBoundaryAPIPathPrefix(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: "/api"}, false)

	// The prefixed CSRF route should issue a usable CSRF pair.
	csrf, csrfCookies, csrfResp := issueCSRF(t, fixture, "/api/auth/csrf")
	assert.NotEmpty(t, csrf)
	assert.Equal(t, http.StatusOK, csrfResp.StatusCode)

	// One prefixed state-changing route is enough to catch prefix mapping regressions.
	_, loginResp := loginThroughBoundary(t, fixture, "/api/login", csrf, csrfCookies)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
}

// Failed refresh does not emit a new session.
func TestBoundaryInvalidRefreshDoesNotSetCookies(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: ""}, false)
	csrf, csrfCookies, _ := issueCSRF(t, fixture, "/auth/csrf")

	// The token is syntactically valid but absent from the refresh-token store.
	invalidRefreshToken, _, _, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), uuid.New())
	require.NoError(t, err)

	// Invalid refresh tokens should fail even when CSRF is valid.
	resp := fixture.proxy(gatewayRequest(http.MethodPost, "/auth/refresh",
		withOrigin(boundaryFrontendOrigin),
		withCookies(append(csrfCookies, "refresh_token="+invalidRefreshToken)),
		withHeader(middleware.CSRFHeaderName, csrf),
	))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// A failed refresh must not set replacement auth cookies.
	cookies := responseSetCookies(resp)
	assert.Empty(t, cookieNames(cookies, "access_token", "refresh_token"))
}

// Raw Cookie headers still reach Gin when API Gateway uses that shape.
func TestBoundaryCookieHeader(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: ""}, false)
	csrf, csrfCookies, _ := issueCSRF(t, fixture, "/auth/csrf")
	sessionCookies, _ := loginThroughBoundary(t, fixture, "/login", csrf, csrfCookies)

	// This uses the raw Cookie header instead of APIGatewayV2HTTPRequest.Cookies.
	resp := fixture.proxy(gatewayRequest(http.MethodGet, "/auth/me",
		withOrigin(boundaryFrontendOrigin),
		withCookieHeader(sessionCookies),
	))

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// CSRF issuance reuses an existing browser token.
func TestBoundaryCSRFTokenReuse(t *testing.T) {
	fixture := newBoundaryFixture(t, boundaryConfigOptions{apiPath: ""}, false)
	csrf, csrfCookies, _ := issueCSRF(t, fixture, "/auth/csrf")

	// A second CSRF request with the cookie should return the same token.
	secondResp := fixture.proxy(gatewayRequest(http.MethodGet, "/auth/csrf",
		withOrigin(boundaryFrontendOrigin),
		withCookies(csrfCookies),
	))
	require.Equal(t, http.StatusOK, secondResp.StatusCode)
	secondCookies := responseSetCookies(secondResp)
	assert.Equal(t, csrf, cookieValue(t, secondCookies, middleware.CSRFCookieName))
}
