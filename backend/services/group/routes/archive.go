package group

import (
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleArchiveGroup(c *gin.Context) {
	h.handleGroupStatus(c, false)
}

func (h *Handler) handleRestoreGroup(c *gin.Context) {
	h.handleGroupStatus(c, true)
}

func (h *Handler) handleGroupStatus(c *gin.Context, active bool) {
	// get param from path
	groupID := c.Param("groupId")
	group, err := h.store.GetGroupByID(groupID)
	if err != nil || group == nil {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}

	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if group.CreateByUser.String() != userID {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}

	// update group status
	if err = h.store.UpdateGroupStatus(groupID, userID, active); err != nil {
		if errors.Is(err, types.ErrGroupUnsettled) {
			utils.WriteError(c, http.StatusConflict, err)
			return
		}
		if errors.Is(err, types.ErrGroupNotExist) {
			utils.WriteError(c, http.StatusNotFound, err)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
