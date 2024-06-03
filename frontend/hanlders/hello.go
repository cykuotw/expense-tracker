package hanlders

import (
	"expense-tracker/frontend/views/hello"

	"github.com/gin-gonic/gin"
)

func HandleHello(c *gin.Context) error {
	return Render(c.Writer, c.Request, hello.Hello("test"))
}
