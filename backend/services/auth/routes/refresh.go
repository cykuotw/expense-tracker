package route

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleRefresh(c *gin.Context) error {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return err
	}

	claims, err := auth.ParseTokenString(refreshToken, "refresh")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return err
	}
	if claims.ID == "" {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return fmt.Errorf("missing refresh token id")
	}

	stored, err := h.refreshStore.GetRefreshTokenByID(claims.ID)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return err
	}
	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return types.ErrInvalidToken
	}
	if stored.TokenHash != auth.HashToken(refreshToken) {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return types.ErrInvalidToken
	}

	if err := h.refreshStore.RevokeRefreshToken(claims.ID); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return err
	}

	accessToken, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	newRefreshToken, newRefreshID, newRefreshExp, err := auth.CreateRefreshJWT(
		[]byte(config.Envs.RefreshJWTSecret), userID,
	)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	if err := h.refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(newRefreshID),
		UserID:    userID,
		TokenHash: auth.HashToken(newRefreshToken),
		ExpiresAt: newRefreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	c.SetCookie(
		"access_token", accessToken,
		int(config.Envs.JWTExpirationInSeconds),
		"/", "localhost", false, true,
	)
	c.SetCookie(
		"refresh_token", newRefreshToken,
		int(config.Envs.RefreshJWTExpirationInSeconds),
		"/", "localhost", false, true,
	)

	utils.WriteJSON(c, http.StatusOK, nil)
	return nil
}
