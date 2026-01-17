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

func TestServiceRegister(t *testing.T) {
	userStore := registerUserStoreMock()
	invitationStore := invitationStoreMock()
	refreshStore := refreshStoreMock()
	handler := NewHandler(userStore, invitationStore, refreshStore)

	t.Run("valid", func(t *testing.T) {
		payload := types.RegisterUserPayload{
			Nickname:  "nickname",
			Firstname: "fname",
			Lastname:  "lname",
			Email:     "adsf@test.com",
			Password:  "longpassword",
			Token:     "test-invite-token",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(marshalled))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/register", handler.handleRegister)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})

	t.Run("invalid payload", func(t *testing.T) {
		payload := types.RegisterUserPayload{
			Nickname:  "nickname",
			Firstname: "fname",
			Lastname:  "lname",
			Email:     "invalid",
			Password:  "pward",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(marshalled))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/register", handler.handleRegister)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
