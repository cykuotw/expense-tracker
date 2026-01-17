package route

import (
	"bytes"
	"encoding/json"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServiceLogin(t *testing.T) {
	userStore := loginUserStoreMock()
	invitationStore := invitationStoreMock()
	refreshStore := refreshStoreMock()
	handler := NewHandler(userStore, invitationStore, refreshStore)

	t.Run("valid-email", func(t *testing.T) {
		payload := types.LoginUserPayload{
			Email:    "adsf@test.com",
			Password: "testpassword",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(marshalled))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.POST("/login", handler.handleLogin)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("invalid payload", func(t *testing.T) {
		payload := types.LoginUserPayload{
			Email:    "invalid",
			Password: "pward",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(marshalled))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.POST("/register", handler.handleRegister)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
