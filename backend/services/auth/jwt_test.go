package auth

import (
	"errors"
	"expense-tracker/backend/config"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExtractJWTClaimValid(t *testing.T) {
	token, err := CreateJWT([]byte(config.Envs.JWTSecret), uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, "/protected", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/protected", func(c *gin.Context) {
		userID, err := ExtractJWTClaim(c, "userID")
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		c.Status(http.StatusOK)
	})
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestExtractJWTClaimExpired(t *testing.T) {
	originalExp := config.Envs.JWTExpirationInSeconds
	config.Envs.JWTExpirationInSeconds = -1
	t.Cleanup(func() {
		config.Envs.JWTExpirationInSeconds = originalExp
	})

	token, err := CreateJWT([]byte(config.Envs.JWTSecret), uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, "/protected", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/protected", func(c *gin.Context) {
		_, err := ExtractJWTClaim(c, "userID")
		assert.True(t, errors.Is(err, types.ErrInvalidToken))
		c.Status(http.StatusOK)
	})
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuthMiddlewareExpired(t *testing.T) {
	originalExp := config.Envs.JWTExpirationInSeconds
	config.Envs.JWTExpirationInSeconds = -1
	t.Cleanup(func() {
		config.Envs.JWTExpirationInSeconds = originalExp
	})

	token, err := CreateJWT([]byte(config.Envs.JWTSecret), uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, "/protected", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/protected", JWTAuthMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
