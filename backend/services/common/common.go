package common

import (
	"expense-tracker/backend/internal/observability"

	"github.com/gin-gonic/gin"
)

type GinHandler func(ctx *gin.Context) error
type GinHandlerMultiErr func(ctx *gin.Context) []error

func Make(h GinHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := h(ctx); err != nil {
			observability.RecordUnhandledHandlerError(ctx, err)
		}
	}
}

func MakeMuitiErr(h GinHandlerMultiErr) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if errors := h(ctx); errors != nil && len(errors) != 0 && errors[0] != nil {
			for _, err := range errors {
				observability.RecordUnhandledHandlerError(ctx, err)
			}
		}
	}
}
