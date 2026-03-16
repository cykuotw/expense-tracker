package extractors

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExtractUserIdFromJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := auth.ExtractJWTClaim(c, "userID")
		if err != nil {
			utils.WriteError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
