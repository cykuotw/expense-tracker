package validation

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
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
			utils.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		if !exist {
			utils.AbortWithError(c, http.StatusBadRequest, types.ErrGroupNotExist)
			return
		}

		c.Next()
	}
}
