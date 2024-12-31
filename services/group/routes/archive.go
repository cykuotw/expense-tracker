package group

import (
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleArchiveGroup(c *gin.Context) {
	// get param from path
	groupID := c.Param("groupId")
	_, err := h.store.GetGroupByID(groupID)
	if err != nil {
		utils.WriteJSON(c, http.StatusBadRequest, err)
		return
	}

	// update group status
	if err = h.store.UpdateGroupStatus(groupID, false); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
