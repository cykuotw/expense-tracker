package extractors

import (
	"expense-tracker/services/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExtractUserIdFromJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := auth.ExtractJWTClaim(c, "userID")
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
