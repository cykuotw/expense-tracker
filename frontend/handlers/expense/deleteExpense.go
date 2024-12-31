package expense

import (
	"encoding/json"
	"expense-tracker/frontend/handlers/common"
	"expense-tracker/types"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetExpenseDelete(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	expenseID := c.Param("expenseId")
	groupId := c.PostForm("groupId")

	// Make a request to delete the expense
	res, err := common.MakeBackendHTTPRequest(http.MethodPut, "/delete_expense/"+expenseID, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		resErr := types.ServerErr{}
		err = json.NewDecoder(res.Body).Decode(&resErr)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		c.Status(http.StatusInternalServerError)
		return fmt.Errorf("%s", resErr.Error)
	}

	c.Header("HX-Redirect", "/group/"+groupId)
	c.Status(http.StatusOK)

	return nil
}
