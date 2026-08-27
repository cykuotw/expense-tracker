package route

import (
	"expense-tracker/backend/config"
	googleAuth "expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/services/common"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store             types.UserStore
	invitationStore   types.InvitationStore
	refreshStore      types.RefreshTokenStore
	registrationStore types.RegistrationStore
	googleService     googleAuth.ServiceContract
	googleVerifier    googleAuth.Verifier
}

func NewHandler(store types.UserStore, invitationStore types.InvitationStore, refreshStore types.RefreshTokenStore, registrationStores ...types.RegistrationStore) *Handler {
	handler := &Handler{
		store:           store,
		invitationStore: invitationStore,
		refreshStore:    refreshStore,
		googleService:   googleAuth.NewService(store),
		googleVerifier:  googleAuth.NewClaimsVerifier(),
	}
	if len(registrationStores) > 0 {
		handler.registrationStore = registrationStores[0]
	}
	return handler
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/auth/csrf", common.Make(h.handleCSRFToken))
	router.GET("/auth/me", common.Make(h.handleAuthMe))
	router.POST("/auth/refresh", common.Make(h.handleRefresh))

	if config.Envs.GoogleOAuthConfigured() {
		handler := h.handleGoogleExchangeInProcess
		if config.Envs.GoogleExchangeModeIs(config.GoogleExchangeUpstreamVerified) {
			handler = h.handleGoogleExchangeUpstreamVerified
		}
		router.POST("/auth/google/exchange", common.Make(handler))

		registerHandler := h.handleGoogleRegisterInProcess
		if config.Envs.GoogleExchangeModeIs(config.GoogleExchangeUpstreamVerified) {
			registerHandler = h.handleGoogleRegisterUpstreamVerified
		}
		router.POST("/auth/google/register", common.Make(registerHandler))
	}

	router.POST("/register", h.handleRegister)
	router.POST("/login", h.handleLogin)
	router.POST("/logout", common.Make(h.handleLogout))
}
