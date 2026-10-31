// Package api assembles the HTTP server from routes, handlers and middleware.
package api

import (
	"fmt"
	"net/http"
	"time"

	"expense-tracker/agent/internal/api/handlers"
	"expense-tracker/agent/internal/api/routes"
	"expense-tracker/internal/localonly"
)

// NewLocalHandler accepts bearer credentials under the explicit local-only boundary.
func NewLocalHandler(application handlers.Application) (http.Handler, error) {
	if err := localonly.CheckRuntime(); err != nil {
		return nil, err
	}
	if application == nil {
		return nil, fmt.Errorf("application required")
	}
	mux := http.NewServeMux()
	routes.Register(mux, handlers.New(application))
	return mux, nil
}

func NewLocalFixtureHandler(application handlers.Application) http.Handler {
	mux := http.NewServeMux()
	routes.RegisterFixture(mux, handlers.New(application))
	return mux
}

func NewHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      70 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}
}
