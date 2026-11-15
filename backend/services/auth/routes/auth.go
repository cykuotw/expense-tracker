package route

import (
	"context"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleAuthMe(c *gin.Context) error {

	userId, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return err
	}

	var user *types.User
	if store, ok := h.store.(interface {
		GetUserByIDContext(context.Context, string) (*types.User, error)
	}); ok {
		user, err = store.GetUserByIDContext(c.Request.Context(), userId)
	} else {
		user, err = h.store.GetUserByID(userId)
	}
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}
	if !user.IsActive {
		utils.WriteError(c, http.StatusForbidden, types.ErrAccountInactive)
		return types.ErrAccountInactive
	}

	utils.WriteJSON(c, http.StatusOK, gin.H{"userID": userId, "role": user.Role})

	return nil
}
