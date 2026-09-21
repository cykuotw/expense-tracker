package frontendreport

import (
	"bytes"
	"expense-tracker/backend/config"
	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/middleware"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const testOrigin = "https://app.example.test"

func TestRenderErrorReportSecurityAndValidation(t *testing.T) {
	router, token, csrf, _ := testRouter(t)
	valid := `{"occurrenceId":"4f64807d-f824-4d3b-9cb9-2cd206548639"}`

	tests := []struct {
		name        string
		body        string
		token       string
		csrf        string
		origin      string
		contentType string
		want        int
	}{
		{name: "accepted", body: valid, token: token, csrf: csrf, origin: testOrigin, contentType: "application/json", want: http.StatusAccepted},
		{name: "authentication required", body: valid, csrf: csrf, origin: testOrigin, contentType: "application/json", want: http.StatusUnauthorized},
		{name: "csrf required", body: valid, token: token, origin: testOrigin, contentType: "application/json", want: http.StatusForbidden},
		{name: "trusted origin required", body: valid, token: token, csrf: csrf, origin: "https://evil.example", contentType: "application/json", want: http.StatusForbidden},
		{name: "json required", body: valid, token: token, csrf: csrf, origin: testOrigin, contentType: "text/plain", want: http.StatusUnsupportedMediaType},
		{name: "uuid required", body: `{"occurrenceId":"raw error text"}`, token: token, csrf: csrf, origin: testOrigin, contentType: "application/json", want: http.StatusBadRequest},
		{name: "unknown fields rejected", body: `{"occurrenceId":"4f64807d-f824-4d3b-9cb9-2cd206548639","stack":"secret"}`, token: token, csrf: csrf, origin: testOrigin, contentType: "application/json", want: http.StatusBadRequest},
		{name: "oversized body rejected", body: `{"occurrenceId":"` + strings.Repeat("a", maxReportBodyBytes) + `"}`, token: token, csrf: csrf, origin: testOrigin, contentType: "application/json", want: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v0/observability/frontend-render-error", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			request.Header.Set("Origin", test.origin)
			request.Header.Set(middleware.CSRFHeaderName, test.csrf)
			if test.csrf != "" {
				request.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: test.csrf})
			}
			if test.token != "" {
				request.AddCookie(&http.Cookie{Name: "access_token", Value: test.token})
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			require.Equal(t, test.want, response.Code)
			require.NotContains(t, response.Body.String(), "secret")
		})
	}
}

func TestRenderErrorReportSuppressesReplayAndUserBurst(t *testing.T) {
	router, token, csrf, logs := testRouter(t)
	report := func(id string) int {
		request := httptest.NewRequest(http.MethodPost, "/api/v0/observability/frontend-render-error", strings.NewReader(`{"occurrenceId":"`+id+`"}`))
		request.Header.Set("Origin", testOrigin)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set(middleware.CSRFHeaderName, csrf)
		request.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: csrf})
		request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response.Code
	}

	require.Equal(t, http.StatusAccepted, report("4f64807d-f824-4d3b-9cb9-2cd206548639"))
	require.Equal(t, http.StatusNoContent, report("4f64807d-f824-4d3b-9cb9-2cd206548639"))
	require.Equal(t, http.StatusNoContent, report("21d8413c-608c-4459-ae67-5619af737a19"))
	require.Equal(t, 1, strings.Count(logs.String(), `"msg":"`+observability.EventFrontendRenderError+`"`))
	require.NotContains(t, logs.String(), "access_token")
}

func testRouter(t *testing.T) (*gin.Engine, string, string, *bytes.Buffer) {
	t.Helper()
	originalOrigin := config.Envs.FrontendOrigin
	config.Envs.FrontendOrigin = testOrigin
	t.Cleanup(func() { config.Envs.FrontendOrigin = originalOrigin })

	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), uuid.New())
	require.NoError(t, err)
	csrf := "test-csrf-token"
	logs := &bytes.Buffer{}
	router := gin.New()
	router.Use(observability.RequestLogging(observability.RequestLoggingConfig{
		Logger: slog.New(slog.NewJSONHandler(logs, nil)),
		Now:    func() time.Time { return time.Unix(0, 0) },
	}))
	group := router.Group("/api/v0")
	group.Use(middleware.CSRFMiddleware(), auth.JWTAuthMiddleware())
	NewHandler().RegisterRoutes(group)
	return router, token, csrf, logs
}
