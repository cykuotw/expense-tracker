// Package routes declares Agent HTTP endpoints and their access middleware.
package routes

import (
	"net/http"

	"expense-tracker/agent/internal/api/handlers"
	"expense-tracker/agent/internal/api/middleware"
)

func Register(mux *http.ServeMux, handler *handlers.Handler) {
	mux.HandleFunc("POST /chat", middleware.WithLocalAccess(handler.HandleChat))
	mux.HandleFunc("GET /chat/conversations", middleware.WithLocalAccess(handler.HandleListConversations))
	mux.HandleFunc("DELETE /chat/conversations/{conversation}", middleware.WithLocalAccess(handler.HandleDeleteConversation))
	mux.HandleFunc("GET /chat/conversations/{conversation}/runs", middleware.WithLocalAccess(handler.HandleListRuns))
	mux.HandleFunc("GET /chat/conversations/{conversation}/runs/{run}/events", middleware.WithLocalAccess(handler.HandleReadRunOutput))
}

func RegisterFixture(mux *http.ServeMux, handler *handlers.Handler) {
	mux.HandleFunc("POST /chat", middleware.WithFixtureAccess(handler.HandleChat))
}
