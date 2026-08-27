package middleware

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type adminUserStore interface {
	GetUserByID(id string) (*types.User, error)
}

func AdminMiddleware(store adminUserStore) gin.HandlerFunc {
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
		if !user.IsActive {
			utils.WriteError(c, http.StatusForbidden, types.ErrAccountInactive)
			c.Abort()
			return
		}

		c.Next()
	}
}
