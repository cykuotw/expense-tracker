package extractors

import (
	"expense-tracker/backend/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExtractExpenseFromStore(store types.ExpenseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		expenseID := c.Param("expenseId")

		expense, err := store.GetExpenseByID(expenseID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Set("expense", expense)
		c.Set("groupID", expense.GroupID.String())

		c.Next()
	}
}

func GetExpenseFromStore(c *gin.Context) (*types.Expense, error) {
	value, exist := c.Get("expense")
	if !exist {
		return nil, types.ErrExpenseNotExist
	}

	expense, ok := value.(*types.Expense)
	if !ok {
		return nil, types.ErrExpenseNotExist
	}

	return expense, nil
}
