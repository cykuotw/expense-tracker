package validation

import (
	"expense-tracker/backend/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ValidateGroupUserPairExist(store types.GroupStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")

		groupID := c.Param("groupId")
		if groupID == "" {
			groupID = c.Query("g")
		}
		if groupID == "" {
			groupID = c.GetString("groupID")
		}

		exist, err := store.CheckGroupUserPairExist(groupID, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
			return
		}
		if !exist {
			c.AbortWithStatusJSON(http.StatusForbidden, types.ErrPermissionDenied)
			return
		}

		c.Next()
	}
}
