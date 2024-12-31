package group

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
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

	// make response
	var response []types.GetGroupListResponse
	for _, group := range groups {
		res := types.GetGroupListResponse{
			ID:          group.ID.String(),
			GroupName:   group.GroupName,
			Description: group.Description,
			Currency:    group.Currency,
		}
		response = append(response, res)
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
