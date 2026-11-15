package utils

import (
	"context"
	"encoding/json"
	"errors"
	"expense-tracker/backend/internal/dbavailability"
	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/types"
	"io"
	"net/http"

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
	if dbavailability.ConnectionFailure(err) || errors.Is(err, dbavailability.ErrUnavailable) || errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusServiceUnavailable
		err = NewDetailedError("Service temporarily unavailable", "service_unavailable", nil)
	}
	if err == nil {
		WriteJSON(c, status, nil)
	} else {
		response := normalizeError(status, err)
		observability.RecordHTTPError(c, status, response.Code, err)
		WriteJSON(c, status, response)
	}
}
