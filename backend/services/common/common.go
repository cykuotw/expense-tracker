package common

import (
	"expense-tracker/backend/internal/observability"

	"github.com/gin-gonic/gin"
)

type GinHandler func(ctx *gin.Context) error

func Make(h GinHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := h(ctx); err != nil {
			observability.RecordUnhandledHandlerError(ctx, err)
		}
	}
}
