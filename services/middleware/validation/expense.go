package validation

import (
	"expense-tracker/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ValidateExpenseExist(store types.ExpenseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		expenseID := c.Param("expenseId")

		exist, err := store.CheckExpenseExistByID(expenseID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
			return
		}
		if !exist {
			c.AbortWithStatusJSON(http.StatusBadRequest, types.ErrExpenseNotExist)
			return
		}

		c.Next()
	}
}
