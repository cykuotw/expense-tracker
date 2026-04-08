package route

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) issueAuthSession(c *gin.Context, user *types.User) error {
	secret := []byte(config.Envs.JWTSecret)
	accessToken, err := auth.CreateJWT(secret, user.ID)
	if err != nil {
		return err
	}

	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), user.ID)
	if err != nil {
		return err
	}

	if err := h.refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		UserID:    user.ID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	setAuthCookies(c, accessToken, refreshToken)
	return nil
}
