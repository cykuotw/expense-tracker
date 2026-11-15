package tracker

import (
	"context"
	"expense-tracker/backend/internal/dbavailability"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseAvailabilityGateAndExplicitRecovery(t *testing.T) {
	policy, err := dbavailability.ParsePolicy("")
	if err != nil {
		t.Fatal(err)
	}
	monitor := dbavailability.New(policy)
	calls := 0
	available := false
	monitor.SetProbe(func(context.Context) error {
		calls++
		if available {
			return nil
		}
		return io.EOF
	})
	router := NewHandler(nil, WithDatabaseAvailability(monitor))
	request := func(method, path string, csrf bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, config.Envs.APIPath+path, nil)
		if csrf {
			req.Header.Set("Origin", config.Envs.FrontendOrigin)
			req.Header.Set("X-CSRF-Token", "fixture")
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "fixture"})
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	if got := request("GET", "/health", false); got.Code != 200 || calls != 0 {
		t.Fatal("health depends on database")
	}
	if got := request("GET", "/auth/csrf", false); got.Code != 200 || calls != 0 {
		t.Fatal("csrf depends on database")
	}
	if got := request("GET", "/auth/me", false); got.Code != 503 {
		t.Fatalf("expected 503: %d", got.Code)
	}
	if got := request("POST", "/service/retry", false); got.Code != 403 || calls != 1 {
		t.Fatal("retry bypassed CSRF")
	}
	available = true
	if got := request("POST", "/service/retry", true); got.Code != 204 || calls != 2 {
		t.Fatalf("manual retry did not recover: %d %d", got.Code, calls)
	}
	if got := request("GET", "/auth/me", false); got.Code != 401 || calls != 2 {
		t.Fatalf("recovery did not reach normal auth: %d", got.Code)
	}
}

func TestNewHandlerRecoversFromPanics(t *testing.T) {
	originalMode := config.Envs.Mode
	config.Envs.Mode = "release"
	t.Cleanup(func() {
		config.Envs.Mode = originalMode
	})

	gin.SetMode(gin.ReleaseMode)
	router := NewHandler(nil)

	engine, ok := router.(*gin.Engine)
	if !ok {
		t.Fatalf("expected *gin.Engine, got %T", router)
	}

	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestNewHandlerRegistersHealthRoute(t *testing.T) {
	originalMode := config.Envs.Mode
	originalAPIPath := config.Envs.APIPath
	config.Envs.Mode = "release"
	config.Envs.APIPath = "/api/v0"
	t.Cleanup(func() {
		config.Envs.Mode = originalMode
		config.Envs.APIPath = originalAPIPath
	})

	router := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
}

func TestImprovementRoutesAreRemoved(t *testing.T) {
	handler := NewHandler(nil)
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/account/ocr-improvement"},
		{http.MethodPatch, "/account/ocr-improvement"},
		{http.MethodPost, "/account/ocr-improvement/purge"},
		{http.MethodPost, "/expense/test-expense/ocr-evidence"},
	} {
		request := httptest.NewRequest(route.method, config.Envs.APIPath+route.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		assert.Equal(t, http.StatusNotFound, response.Code, route.path)
	}
}

func TestNewHTTPServerAppliesTimeouts(t *testing.T) {
	handler := http.NewServeMux()
	server := NewHTTPServer("127.0.0.1:8000", handler)

	assert.Equal(t, "127.0.0.1:8000", server.Addr)
	assert.Same(t, handler, server.Handler)
	assert.Equal(t, readHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, readTimeout, server.ReadTimeout)
	assert.Equal(t, writeTimeout, server.WriteTimeout)
	assert.Equal(t, idleTimeout, server.IdleTimeout)
}
