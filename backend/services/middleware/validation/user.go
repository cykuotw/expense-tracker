package validation

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ValidateUserExist(store types.UserStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")

		exist, err := store.CheckUserExistByID(userID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			c.Abort()
			return
		}
		if !exist {
			utils.WriteError(c, http.StatusBadRequest, types.ErrUserNotExist)
			c.Abort()
			return
		}

		c.Next()
	}
}
