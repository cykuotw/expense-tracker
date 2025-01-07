package validation

import (
	"expense-tracker/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ValidateGroupExist(store types.GroupStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("groupId")
		if groupID == "" {
			groupID = c.Query("g")
		}
		if groupID == "" {
			groupID = c.GetString("groupID")
		}

		exist, err := store.CheckGroupExistById(groupID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
			return
		}
		if !exist {
			c.AbortWithStatusJSON(http.StatusBadRequest, types.ErrGroupNotExist)
			return
		}

		c.Next()
	}
}
