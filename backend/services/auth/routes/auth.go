package route

import (
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

	user, err := h.store.GetUserByID(userId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	utils.WriteJSON(c, http.StatusOK, gin.H{"userID": userId, "role": user.Role})

	return nil
}
