package group

import (
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetArchivedGroups(c *gin.Context) {
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidGroupCursor)
		return
	}
	store, ok := h.store.(types.GroupLifecycleStore)
	if !ok {
		utils.WriteError(c, http.StatusInternalServerError, types.ErrInvalidAction)
		return
	}
	page, err := store.GetArchivedGroups(userID, c.Query("cursor"), limit)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, types.ErrInvalidGroupCursor) {
			status = http.StatusBadRequest
		}
		utils.WriteError(c, status, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, page)
}
