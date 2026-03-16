package route

import (
	"expense-tracker/backend/services/middleware"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleCSRFToken(c *gin.Context) error {
	token, err := middleware.IssueCSRFToken(c)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	utils.WriteJSON(c, http.StatusOK, gin.H{"csrfToken": token})
	return nil
}
