package route

import (
	"expense-tracker/services/common"
	"expense-tracker/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store types.UserStore
}

func NewHandler(store types.UserStore) *Handler {
	initThirdParty()

	return &Handler{
		store: store,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/:provider", common.Make(h.handleThirdParty))
	router.GET("/auth/:provider/callback", common.Make(h.handleThirdPartyCallback))

	router.GET("/auth/me", common.Make(h.handleAuthMe))
}
