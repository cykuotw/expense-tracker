package common

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
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

func Render(w http.ResponseWriter, r *http.Request, c templ.Component) error {
	return c.Render(r.Context(), w)
}
