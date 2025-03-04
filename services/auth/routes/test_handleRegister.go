package route

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

func TestServiceRegister(t *testing.T) {
	userStore := &mockStoreRegister{}
	handler := NewHandler(userStore)

	t.Run("valid", func(t *testing.T) {
		payload := types.RegisterUserPayload{
			Nickname:  "nickname",
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

type mockStoreRegister struct{}

func (m *mockStoreRegister) GetUserByEmail(email string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}
func (m *mockStoreRegister) GetUserByUsername(username string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}
func (m *mockStoreRegister) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockStoreRegister) CreateUser(user types.User) error {
	return nil
}
func (m *mockStoreRegister) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockStoreRegister) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreRegister) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreRegister) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockStoreRegister) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}
