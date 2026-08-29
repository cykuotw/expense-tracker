package group

import (
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetGroupMember(c *gin.Context) {
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

	// check requester belongs to the group
	exist, err := h.store.CheckGroupUserPairExist(groupId, userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !exist {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}

	// get members of the group
	users, err := h.store.GetGroupMemberByGroupID(groupId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// make response
	members := make([]types.GroupMember, 0, len(users))
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

	utils.WriteJSON(c, http.StatusOK, members)
}

func (h *Handler) handleUpdateGroupMember(c *gin.Context) {
	// get payload
	var payload types.UpdateGroupMemberPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// validate payload
	if payload.Action != "add" && payload.Action != "delete" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidAction)
		return
	}
	// get requester user id from jwt
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	group, err := h.store.GetGroupByID(payload.GroupID)
	if err != nil {
		if errors.Is(err, types.ErrGroupNotExist) {
			utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if group == nil || group.CreateByUser.String() != userID {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}

	exist, err := h.userStore.CheckUserExistByID(payload.UserID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !exist {
		utils.WriteError(c, http.StatusBadRequest, types.ErrUserNotExist)
		return
	}

	if payload.Action == "delete" && payload.UserID == group.CreateByUser.String() {
		utils.WriteError(c, http.StatusBadRequest, types.ErrProtectedGroupMember)
		return
	}
	if payload.Action == "delete" {
		members, err := h.store.GetGroupMemberByGroupID(payload.GroupID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		if len(members) <= 1 {
			utils.WriteError(c, http.StatusBadRequest, types.ErrProtectedGroupMember)
			return
		}
	}

	// update group member
	err = h.store.UpdateGroupMember(payload.Action, payload.UserID, payload.GroupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}

func (h *Handler) handleGetRelatedMember(c *gin.Context) {
	groupId := c.Query("g")
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	exist, err := h.store.CheckGroupUserPairExist(groupId, userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !exist {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}

	members, err := h.store.GetRelatedUser(userID, groupId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if members == nil {
		members = []*types.RelatedMember{}
	}

	utils.WriteJSON(c, http.StatusOK, members)
}
