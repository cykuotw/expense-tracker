package validation

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ValidateExpenseExist(store types.ExpenseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		expenseID := c.Param("expenseId")

		exist, err := store.CheckExpenseExistByID(expenseID)
		if err != nil {
			utils.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		if !exist {
			utils.AbortWithError(c, http.StatusBadRequest, types.ErrExpenseNotExist)
			return
		}

		c.Next()
	}
}
