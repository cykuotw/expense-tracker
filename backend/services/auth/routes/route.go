package route

import (
	"expense-tracker/backend/services/common"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store           types.UserStore
	invitationStore types.InvitationStore
	refreshStore    types.RefreshTokenStore
}

func NewHandler(store types.UserStore, invitationStore types.InvitationStore, refreshStore types.RefreshTokenStore) *Handler {
	initThirdParty()

	return &Handler{
		store:           store,
		invitationStore: invitationStore,
		refreshStore:    refreshStore,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/:provider", common.Make(h.handleThirdParty))
	router.GET("/auth/:provider/callback", common.Make(h.handleThirdPartyCallback))

	router.GET("/auth/me", common.Make(h.handleAuthMe))
	router.POST("/auth/refresh", common.Make(h.handleRefresh))

	router.POST("/register", h.handleRegister)
	router.POST("/login", h.handleLogin)
	router.POST("/logout", common.Make(h.handleLogout))
}
