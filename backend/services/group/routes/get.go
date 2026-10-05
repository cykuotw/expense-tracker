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
	currencySettings := types.GroupCurrencySettings{}
	if settingsStore, ok := h.store.(types.GroupCurrencySettingsStore); ok {
		currencySettings, err = settingsStore.GetGroupCurrencySettings(groupId, userID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	lifecycle, err := types.ReadGroupLifecycle(h.store, group, userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	response := types.GetGroupResponse{
		GroupLifecycle:   lifecycle,
		CurrentUserID:    userID,
		GroupName:        group.GroupName,
		Description:      group.Description,
		Currency:         group.Currency,
		CurrencySettings: currencySettings,
		CurrencyEditable: currencyEditable && lifecycle.IsActive,
		DetailsEditable:  group.CreateByUser.String() == userID && lifecycle.IsActive,
		GroupType:        group.GroupType,
		Members:          groupMembersForUser(users, userID),
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
