package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type validationPayload struct {
	Email string `validate:"required,email"`
}

func TestWriteErrorMapsTypedErrorToCode(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)

	WriteError(c, http.StatusForbidden, types.ErrPermissionDenied)

	var body APIErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "permission denied", body.Error)
	assert.Equal(t, "permission_denied", body.Code)
}

func TestWriteErrorSanitizesInternalErrors(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(observability.RequestLogging(observability.RequestLoggingConfig{
		Logger:       observability.NewLogger("release", &output),
		NewRequestID: func() string { return "request-id" },
	}))
	router.GET("/failure", func(c *gin.Context) {
		WriteError(c, http.StatusInternalServerError, errors.New("database-password-secret"))
	})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/failure", nil))

	var body APIErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "internal server error", body.Error)
	assert.Equal(t, "internal_error", body.Code)
	assert.Equal(t, 1, bytes.Count(output.Bytes(), []byte(observability.EventUnexpectedHTTPError)))
	assert.Contains(t, output.String(), `"error_category":"unclassified_error"`)
	assert.Contains(t, output.String(), `"diagnostic_message":"unexpected internal failure"`)
	assert.NotContains(t, output.String(), "database-password-secret")
}

func TestNewValidationErrorIncludesDetails(t *testing.T) {
	err := Validate.Struct(validationPayload{})
	validationErrs := err.(validator.ValidationErrors)

	apiErr := normalizeError(http.StatusBadRequest, NewValidationError(validationErrs))

	assert.Equal(t, "invalid payload", apiErr.Error)
	assert.Equal(t, "invalid_payload", apiErr.Code)

	details, ok := apiErr.Details.([]ValidationErrorDetail)
	if assert.True(t, ok) && assert.Len(t, details, 1) {
		assert.Equal(t, "email", details[0].Field)
		assert.Equal(t, "required", details[0].Code)
		assert.Equal(t, "is required", details[0].Message)
	}
}

func TestWriteErrorDetectsMalformedJSON(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)

	WriteError(c, http.StatusBadRequest, &json.SyntaxError{Offset: 7})

	var body APIErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "invalid json payload", body.Error)
	assert.Equal(t, "invalid_json", body.Code)
	assert.NotNil(t, body.Details)
}
