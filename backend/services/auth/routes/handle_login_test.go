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
		var response struct {
			Error   string `json:"error"`
			Code    string `json:"code"`
			Details []struct {
				Field string `json:"field"`
				Code  string `json:"code"`
			} `json:"details"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "invalid payload", response.Error)
		assert.Equal(t, "invalid_payload", response.Code)
		if assert.NotEmpty(t, response.Details) {
			fields := make([]string, 0, len(response.Details))
			for _, detail := range response.Details {
				fields = append(fields, detail.Field)
			}
			assert.Contains(t, fields, "email")
		}
	})

	t.Run("external user password login is rejected", func(t *testing.T) {
		userStore := &baseAuthUserStore{
			GetUserByEmailFn: func(email string) (*types.User, error) {
				return &types.User{
					Email:          email,
					ExternalType:   "google",
					PasswordHashed: "unused",
				}, nil
			},
		}
		handler := NewHandler(userStore, invitationStoreMock(), refreshStoreMock())

		payload := types.LoginUserPayload{
			Email:    "google-user@example.com",
			Password: "anything",
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

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var response struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "invalid_credentials", response.Code)
	})
}
