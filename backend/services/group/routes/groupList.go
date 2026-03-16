package group

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetGroupList(c *gin.Context) {
	// get user id from jwt
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get group id list where user id as member
	groups, err := h.store.GetGroupListByUser(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusOK, groups)
}
