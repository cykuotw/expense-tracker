package validation

import (
	"expense-tracker/backend/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ValidateBalanceExist(store types.ExpenseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		balanceId := c.Param("balanceId")
		if balanceId == "" {
			balanceId = c.Query("g")
		}
		if balanceId == "" {
			balanceId = c.GetString("balanceId")
		}

		exist, err := store.CheckBalanceExistByID(balanceId)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
			return
		}
		if !exist {
			c.AbortWithStatusJSON(http.StatusBadRequest, types.ErrBalanceNotExist)
			return
		}

		c.Next()
	}
}
