package agenttools

import (
	"fmt"

	"expense-tracker/backend/config"
	"expense-tracker/internal/localonly"
	"github.com/gin-gonic/gin"
)

// NewLocalHandler is selected only by standalone startup, never NewHandler or
// Lambda startup. It skips IAM only; JWT/domain authorization is shared.
func NewLocalHandler(users Users, groups Groups, balances Balances) (*Handler, error) {
	if err := localonly.CheckRuntime(); err != nil {
		return nil, err
	}
	if config.BuildMode == "release" || config.Envs.Mode == "release" {
		return nil, fmt.Errorf("local agent tools are unavailable in release builds")
	}
	return &Handler{users: users, groups: groups, balances: balances, local: true}, nil
}

func localBoundary(c *gin.Context) bool {
	if !localonly.RequestAllowed(c.Request) || len(c.Request.Header.Values("Cookie")) != 0 || len(c.Request.Header.Values("Authorization")) != 0 {
		return false
	}
	// The local client is server-to-server, without browser-origin authority.
	return len(c.Request.Header.Values("Origin")) == 0 && c.Request.Method == "GET"
}
