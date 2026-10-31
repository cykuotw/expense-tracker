package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/agent/internal/credentials"
)

func TestLocalAccessPassesCredentialsThroughContext(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8090/chat", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Authorization", "Bearer opaque-access")
	called := false
	handler := WithLocalAccess(func(writer http.ResponseWriter, request *http.Request) {
		called = true
		if credentials.AccessToken(request.Context()) != "opaque-access" {
			t.Fatal("access token was not passed to the application context")
		}
		writer.WriteHeader(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	handler(response, request)
	if !called || response.Code != http.StatusNoContent || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("valid local request was not forwarded")
	}
}

func TestLocalAccessRejectsBeforeCallingApplication(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*http.Request)
		status int
	}{
		{"missing credentials", func(r *http.Request) { r.Header.Del("Authorization") }, http.StatusUnauthorized},
		{"remote peer", func(r *http.Request) { r.RemoteAddr = "192.0.2.1:12345" }, http.StatusForbidden},
		{"forwarded request", func(r *http.Request) { r.Header.Set("Forwarded", "for=127.0.0.1") }, http.StatusForbidden},
		{"cookie credentials", func(r *http.Request) { r.Header.Set("Cookie", "session=synthetic") }, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8090/chat", nil)
			request.RemoteAddr = "127.0.0.1:12345"
			request.Header.Set("Authorization", "Bearer opaque-access")
			test.mutate(request)
			response := httptest.NewRecorder()
			WithLocalAccess(func(http.ResponseWriter, *http.Request) {
				t.Fatal("rejected request reached the application")
			})(response, request)
			if response.Code != test.status {
				t.Fatalf("got status %d, want %d", response.Code, test.status)
			}
		})
	}
}
