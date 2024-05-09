package utils

import (
	"encoding/json"
	"expense-tracker/types"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

func ParseJSON(c *gin.Context, payload any) error {
	err := json.NewDecoder(c.Request.Body).Decode(payload)
	if err == io.EOF {
		return types.ErrEmptyRequestBody
	}
	if err != nil {
		return err
	}
	return nil
}

func WriteJSON(c *gin.Context, status int, obj any) {
	c.Header("Content-Type", "application/json")
	c.JSON(status, obj)
}

func WriteError(c *gin.Context, status int, err error) {
	WriteJSON(c, status, map[string]string{"error": err.Error()})
}
