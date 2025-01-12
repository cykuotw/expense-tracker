package expense

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleSettleExpense(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	groupId := c.Query("g")

	fmt.Println("token:", token)
	fmt.Println("groupId:", groupId)

	c.Header("HX-Redirect", "/group/"+groupId)
	c.Status(http.StatusOK)

	return nil
}
