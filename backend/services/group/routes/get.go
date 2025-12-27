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
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// get members of the group
	users, err := h.store.GetGroupMemberByGroupID(groupId)
	if err != nil {
		utils.WriteJSON(c, http.StatusInternalServerError, err)
		return
	}

	// make response
	var members []types.GroupMember
	var currUser types.GroupMember
	for _, user := range users {
		if user.ID.String() == userID {
			currUser = types.GroupMember{
				UserID:   user.ID.String(),
				Username: user.Username,
			}
			continue
		}

		member := types.GroupMember{
			UserID:   user.ID.String(),
			Username: user.Username,
		}
		members = append(members, member)
	}
	members = append(members, currUser)

	response := types.GetGroupResponse{
		GroupName:   group.GroupName,
		Description: group.Description,
		Currency:    group.Currency,
		Members:     members,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
