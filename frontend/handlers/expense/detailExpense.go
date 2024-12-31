package expense

import (
	"encoding/json"
	"expense-tracker/frontend/handlers/common"
	"expense-tracker/frontend/views/index"
	"expense-tracker/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetExpenseDetail(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	expenseID := c.Param("expenseId")

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/expense/"+expenseID, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	resPayload := types.ExpenseResponse{}
	err = json.NewDecoder(res.Body).Decode(&resPayload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	return common.Render(c.Writer, c.Request, index.ExpenseDetail(resPayload))
}
