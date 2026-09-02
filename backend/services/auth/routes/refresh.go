package route

import (
	"errors"
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

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return types.ErrInvalidToken
	}
	user, err := h.store.GetUserByID(claims.UserID)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return err
	}
	if !user.IsActive {
		if revokeErr := h.refreshStore.RevokeRefreshTokenFamily(claims.ID); revokeErr != nil {
			utils.WriteError(c, http.StatusInternalServerError, revokeErr)
			return revokeErr
		}
		utils.WriteError(c, http.StatusForbidden, types.ErrAccountInactive)
		return types.ErrAccountInactive
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

	if err := h.refreshStore.RotateRefreshToken(claims.ID, auth.HashToken(refreshToken), types.RefreshToken{
		ID:        uuid.MustParse(newRefreshID),
		UserID:    userID,
		TokenHash: auth.HashToken(newRefreshToken),
		ExpiresAt: newRefreshExp,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		if errors.Is(err, types.ErrInvalidToken) {
			utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
			return err
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	setAuthCookies(c, accessToken, newRefreshToken)

	utils.WriteJSON(c, http.StatusOK, nil)
	return nil
}
