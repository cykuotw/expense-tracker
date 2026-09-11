package group

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetGroup(c *gin.Context) {
	// get group id
	groupId := c.Param("groupid")
	if groupId == "" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrGroupNotExist)
		return
	}

	// get user id from jwt
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get group detail by id
	group, err := h.store.GetGroupByIDAndUser(groupId, userID)
	if err != nil {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}

	// get members of the group
	users, err := h.store.GetGroupMemberByGroupID(groupId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	currencyEditable, err := h.store.CanEditGroupCurrency(groupId, userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	response := types.GetGroupResponse{
		CurrentUserID:    userID,
		GroupName:        group.GroupName,
		Description:      group.Description,
		Currency:         group.Currency,
		CurrencyEditable: currencyEditable,
		DetailsEditable:  group.CreateByUser.String() == userID,
		GroupType:        group.GroupType,
		Members:          groupMembersForUser(users, userID),
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
