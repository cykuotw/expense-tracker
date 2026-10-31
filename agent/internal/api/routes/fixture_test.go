package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/agent/internal/api/handlers"
)

func TestValidationAndFixtureIsolation(t *testing.T) {
	testCases := []struct {
		name   string
		body   string
		mutate func(*http.Request)
		status int
	}{
		{
			name:   "invalid json",
			body:   "{}",
			status: http.StatusBadRequest,
		},
		{
			name:   "trailing json",
			body:   balanceRequestBody + "{}",
			status: http.StatusBadRequest,
		},
		{
			name:   "unknown field",
			body:   `{"conversationId":"one","messageId":"one","text":"x","actor":"forged"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "duplicate key",
			body:   `{"conversationId":"one","conversationId":"two","messageId":"one","text":"x"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "oversized",
			body:   strings.Repeat("x", 17<<10),
			status: http.StatusRequestEntityTooLarge,
		},
		{
			name: "remote peer",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.RemoteAddr = "192.0.2.1:1234"
			},
			status: http.StatusForbidden,
		},
		{
			name: "foreign host",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Host = "foreign.example"
				request.URL.Host = "foreign.example"
			},
			status: http.StatusForbidden,
		},
		{
			name: "origin",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Origin", "https://foreign.example")
			},
			status: http.StatusForbidden,
		},
		{
			name: "forged identity",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("X-Actor-ID", "admin")
			},
			status: http.StatusForbidden,
		},
		{
			name: "bearer",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Authorization", "Bearer forged")
			},
			status: http.StatusForbidden,
		},
		{
			name: "cookie",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Cookie", "session=forged")
			},
			status: http.StatusForbidden,
		},
		{
			name: "content type",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Content-Type", "text/plain")
			},
			status: http.StatusUnsupportedMediaType,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := localRequest(testCase.body)
			if testCase.mutate != nil {
				testCase.mutate(request)
			}
			writer := httptest.NewRecorder()
			mux := http.NewServeMux()
			RegisterFixture(mux, handlers.New(chatApplication{}))
			mux.ServeHTTP(writer, request)
			if writer.Code != testCase.status {
				t.Fatalf("status %d want %d: %s", writer.Code, testCase.status, writer.Body.String())
			}
			if strings.Contains(writer.Body.String(), "RUN_STARTED") {
				t.Fatal("invalid request started model work")
			}
		})
	}
}
