package user

import (
	"bytes"
	"encoding/json"
	"expense-tracker/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUserServiceRegister(t *testing.T) {
	userStore := &mockUserStore{}
	handler := NewHandler(userStore)

	t.Run("valid", func(t *testing.T) {
		payload := types.RegisterUserPayload{
			Username:  "uname",
			Firstname: "fname",
			Lastname:  "lname",
			Email:     "adsf@test.com",
			Password:  "longpassword",
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

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("invalid payload", func(t *testing.T) {
		payload := types.RegisterUserPayload{
			Username:  "uname",
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

type mockUserStore struct{}

func (m *mockUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}
func (m *mockUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}
func (m *mockUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockUserStore) CreateUser(user types.User) error {
	return nil
}
