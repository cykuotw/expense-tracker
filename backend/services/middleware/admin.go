package middleware

import (
	"context"
	"errors"
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

type activeUserStore interface {
	GetUserByID(id string) (*types.User, error)
}

func loadUser(c *gin.Context, store activeUserStore, id string) (*types.User, error) {
	if contextual, ok := store.(interface {
		GetUserByIDContext(context.Context, string) (*types.User, error)
	}); ok {
		return contextual.GetUserByIDContext(c.Request.Context(), id)
	}
	return store.GetUserByID(id)
}

func ActiveUserMiddleware(store activeUserStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := auth.ExtractJWTClaim(c, "userID")
		if err != nil {
			utils.WriteError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		user, err := loadUser(c, store, userID)
		if err != nil && !errors.Is(err, types.ErrUserNotExist) {
			utils.WriteError(c, http.StatusInternalServerError, err)
			c.Abort()
			return
		}
		if err != nil || user == nil || !user.IsActive {
			utils.WriteError(c, http.StatusForbidden, types.ErrAccountInactive)
			c.Abort()
			return
		}

		c.Next()
	}
}

func AdminMiddleware(store adminUserStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := auth.ExtractJWTClaim(c, "userID")
		if err != nil {
			utils.WriteError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		user, err := loadUser(c, store, userID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
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
