package group

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func isValidGroupType(groupType string) bool {
	switch groupType {
	case "trip", "home", "family", "friends", "event", "other":
		return true
	default:
		return false
	}
}

func (h *Handler) handleUpdateGroup(c *gin.Context) {
	var payload types.UpdateGroupPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if !isValidGroupType(payload.GroupType) {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidAction)
		return
	}
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	group, err := h.store.GetGroupByID(c.Param("groupid"))
	if err != nil || group.CreateByUser.String() != userID {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}
	group.GroupName, group.Description, group.Currency, group.GroupType = payload.GroupName, payload.Description, payload.Currency, payload.GroupType
	if err := h.store.UpdateGroup(*group); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, nil)
}
