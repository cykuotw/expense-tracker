package user

import (
	"context"
	"errors"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	googleAuth "expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AccountStore interface {
	GetUserByID(id string) (*types.User, error)
	UpdateOwnProfile(userID string, payload types.UpdateOwnProfilePayload) error
	ChangeOwnPassword(userID string, currentPassword string, newPassword string, preserveRefreshID string) error
	LinkGoogleIdentity(ctx context.Context, userID string, currentPassword string, externalID string, verifiedEmail string, preserveRefreshID string) error
}

type HandlerProtected struct {
	store          AccountStore
	googleVerifier googleAuth.Verifier
}

func NewProtectedHandler(store AccountStore) *HandlerProtected {
	return &HandlerProtected{
		store:          store,
		googleVerifier: googleAuth.NewClaimsVerifier(),
	}
}

func (h *HandlerProtected) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/account", h.handleAccount)
	router.PATCH("/account", h.handleUpdateProfile)
	router.PATCH("/account/password", h.handleChangePassword)
	if config.Envs.GoogleOAuthConfigured() {
		googleLinkHandler := h.handleGoogleLinkInProcess
		if config.Envs.GoogleExchangeModeIs(config.GoogleExchangeUpstreamVerified) {
			googleLinkHandler = h.handleGoogleLinkUpstreamVerified
		}
		router.POST("/account/google/link", googleLinkHandler)
	}
}

func accountResponse(user *types.User) types.AccountResponse {
	return types.AccountResponse{
		Nickname:              user.Nickname,
		Firstname:             user.Firstname,
		Lastname:              user.Lastname,
		Email:                 user.Email,
		GoogleConnected:       user.ExternalType == "google",
		PasswordChangeAllowed: user.HasLocalPassword,
	}
}

func accountUserID(c *gin.Context) (string, bool) {
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return "", false
	}
	return userID, true
}

func writeAccountError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, types.ErrCurrentPasswordIncorrect):
		utils.WriteError(c, http.StatusUnauthorized, err)
	case errors.Is(err, types.ErrPasswordChangeUnavailable),
		errors.Is(err, types.ErrPasswordUnchanged),
		errors.Is(err, types.ErrGoogleLinkEmailMismatch),
		errors.Is(err, types.ErrGoogleAccountConflict),
		errors.Is(err, types.ErrGoogleAlreadyConnected),
		errors.Is(err, types.ErrGoogleLinkUnavailable):
		utils.WriteError(c, http.StatusConflict, err)
	case errors.Is(err, types.ErrInvalidProfile):
		utils.WriteError(c, http.StatusBadRequest, err)
	case errors.Is(err, types.ErrInvalidPasswordLength):
		utils.WriteError(c, http.StatusBadRequest, err)
	case errors.Is(err, types.ErrAccountInactive):
		utils.WriteError(c, http.StatusForbidden, err)
	case errors.Is(err, types.ErrUserNotExist):
		utils.WriteError(c, http.StatusNotFound, err)
	default:
		utils.WriteError(c, http.StatusInternalServerError, err)
	}
}

func (h *HandlerProtected) handleAccount(c *gin.Context) {
	userID, ok := accountUserID(c)
	if !ok {
		return
	}
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		writeAccountError(c, err)
		return
	}
	if !user.IsActive {
		writeAccountError(c, types.ErrAccountInactive)
		return
	}
	utils.WriteJSON(c, http.StatusOK, accountResponse(user))
}

func (h *HandlerProtected) handleUpdateProfile(c *gin.Context) {
	var payload types.UpdateOwnProfilePayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	payload.Firstname = strings.TrimSpace(payload.Firstname)
	payload.Lastname = strings.TrimSpace(payload.Lastname)
	payload.Nickname = strings.TrimSpace(payload.Nickname)
	if payload.Firstname == "" || payload.Lastname == "" {
		writeAccountError(c, types.ErrInvalidProfile)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(err.(validator.ValidationErrors)))
		return
	}
	userID, ok := accountUserID(c)
	if !ok {
		return
	}
	if err := h.store.UpdateOwnProfile(userID, payload); err != nil {
		writeAccountError(c, err)
		return
	}
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		writeAccountError(c, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, accountResponse(user))
}

func currentRefreshID(c *gin.Context, userID string) string {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		return ""
	}
	claims, err := auth.ParseTokenString(refreshToken, "refresh")
	if err != nil || claims.UserID != userID {
		return ""
	}
	return claims.ID
}

func (h *HandlerProtected) handleChangePassword(c *gin.Context) {
	var payload types.ChangeOwnPasswordPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(err.(validator.ValidationErrors)))
		return
	}
	if len(payload.CurrentPassword) > 72 || len(payload.NewPassword) < 8 || len(payload.NewPassword) > 72 {
		writeAccountError(c, types.ErrInvalidPasswordLength)
		return
	}
	userID, ok := accountUserID(c)
	if !ok {
		return
	}
	if err := h.store.ChangeOwnPassword(
		userID,
		payload.CurrentPassword,
		payload.NewPassword,
		currentRefreshID(c, userID),
	); err != nil {
		writeAccountError(c, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, gin.H{"otherSessionsRevoked": true})
}
