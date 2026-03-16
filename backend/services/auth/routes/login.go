package route

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func (h *Handler) handleLogin(c *gin.Context) {
	// get json payload
	var payload types.LoginUserPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	if err := utils.Validate.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(errors))
		return
	}

	var user *types.User
	var err error
	user, err = h.store.GetUserByEmail(payload.Email)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	if !auth.ValidatePassword(user.PasswordHashed, payload.Password) {
		utils.WriteError(c, http.StatusBadRequest, types.ErrPasswordNotMatch)
		return
	}

	secret := []byte(config.Envs.JWTSecret)
	accessToken, err := auth.CreateJWT(secret, user.ID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), user.ID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if err := h.refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		UserID:    user.ID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	setAuthCookies(c, accessToken, refreshToken)
	utils.WriteJSON(c, http.StatusOK, nil)
}
