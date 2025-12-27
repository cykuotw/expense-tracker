package route

import (
	"expense-tracker/backend/services/common"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store           types.UserStore
	invitationStore types.InvitationStore
}

func NewHandler(store types.UserStore, invitationStore types.InvitationStore) *Handler {
	initThirdParty()

	return &Handler{
		store:           store,
		invitationStore: invitationStore,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/:provider", common.Make(h.handleThirdParty))
	router.GET("/auth/:provider/callback", common.Make(h.handleThirdPartyCallback))

	router.GET("/auth/me", common.Make(h.handleAuthMe))

	router.POST("/register", h.handleRegister)
	router.POST("/login", h.handleLogin)
	router.POST("/logout", common.Make(h.handleLogout))
}
