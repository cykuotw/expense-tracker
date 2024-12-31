package group

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleCreateGroup(c *gin.Context) {
	// get payload
	var payload types.CreateGroupPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// get user id from jwt
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// check if user id exist
	user, err := h.userStore.GetUserByID(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	if payload.GroupName == "" {
		payload.GroupName = "Default Group Name"
	}

	group := types.Group{
		ID:           uuid.New(),
		GroupName:    payload.GroupName,
		Description:  payload.Description,
		CreateTime:   time.Now(),
		IsActive:     true,
		Currency:     payload.Currency,
		CreateByUser: user.ID,
	}

	err = h.store.CreateGroup(group)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, map[string]string{"groupId": group.ID.String()})
}
