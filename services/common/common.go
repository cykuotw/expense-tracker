package common

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

type GinHandler func(ctx *gin.Context) error
type GinHandlerMultiErr func(ctx *gin.Context) []error

func Make(h GinHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := h(ctx); err != nil {
			slog.Error("HTTP handler error", "err", err, "path", ctx.Request.URL.Path)
		}
	}
}

func MakeMuitiErr(h GinHandlerMultiErr) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if errors := h(ctx); errors != nil && len(errors) != 0 && errors[0] != nil {
			for _, err := range errors {

				slog.Error("HTTP handler error", "err", err, "path", ctx.Request.URL.Path)
			}
		}
	}
}
