package validation

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
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
			utils.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		if !exist {
			utils.AbortWithError(c, http.StatusBadRequest, types.ErrBalanceNotExist)
			return
		}

		c.Next()
	}
}
