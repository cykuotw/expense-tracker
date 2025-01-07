package extractors

import (
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExtractExpensePayload() gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload types.ExpensePayload
		if err := utils.ParseJSON(c, &payload); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.Set("expensePayload", payload)
		c.Set("groupID", payload.GroupID)
		c.Next()
	}
}

func GetExpensePayload(c *gin.Context) (types.ExpensePayload, error) {
	value, exist := c.Get("expensePayload")
	if !exist {
		return types.ExpensePayload{}, types.ErrExpenseNotExist
	}

	payload, ok := value.(types.ExpensePayload)
	if !ok {
		return types.ExpensePayload{}, types.ErrExpenseNotExist
	}

	return payload, nil
}

func ExtractExpenseUpdatePayload() gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload types.ExpenseUpdatePayload
		if err := utils.ParseJSON(c, &payload); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.Set("expensePayload", payload)
		c.Set("groupID", payload.GroupID.String())
		c.Next()
	}
}

func GetExpenseUpdatePayload(c *gin.Context) (types.ExpenseUpdatePayload, error) {
	value, exist := c.Get("expensePayload")
	if !exist {
		return types.ExpenseUpdatePayload{}, types.ErrExpenseNotExist
	}

	payload, ok := value.(types.ExpenseUpdatePayload)
	if !ok {
		return types.ExpenseUpdatePayload{}, types.ErrExpenseNotExist
	}

	return payload, nil
}
