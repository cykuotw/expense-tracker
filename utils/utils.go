package utils

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

func ParseJSON(c *gin.Context, payload any) error {
	if c.Request.Body == nil {
		return fmt.Errorf("missing request body")
	}
	return json.NewDecoder(c.Request.Body).Decode(payload)
}

func WriteJSON(c *gin.Context, status int, obj any) {
	c.Header("Content-Type", "application/json")
	c.JSON(status, obj)
}

func WriteError(c *gin.Context, status int, err error) {
	WriteJSON(c, status, map[string]string{"error": err.Error()})
}
