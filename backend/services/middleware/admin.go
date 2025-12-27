package middleware

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminMiddleware(store types.UserStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := auth.ExtractJWTClaim(c, "userID")
		if err != nil {
			utils.WriteError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		user, err := store.GetUserByID(userID)
		if err != nil {
			utils.WriteError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		if user.Role != "admin" {
			utils.WriteError(c, http.StatusForbidden, fmt.Errorf("requires admin permission"))
			c.Abort()
			return
		}

		c.Next()
	}
}
