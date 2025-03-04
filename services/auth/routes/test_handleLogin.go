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
	"golang.org/x/crypto/bcrypt"
)

func TestServiceLogin(t *testing.T) {
	userStore := &mockStoreLogin{}
	handler := NewHandler(userStore)

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

type mockStoreLogin struct{}

func (m *mockStoreLogin) GetUserByEmail(email string) (*types.User, error) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	user := types.User{
		PasswordHashed: string(hash),
	}
	return &user, nil
}
func (m *mockStoreLogin) GetUserByUsername(username string) (*types.User, error) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	user := types.User{
		PasswordHashed: string(hash),
	}
	return &user, nil
}
func (m *mockStoreLogin) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockStoreLogin) CreateUser(user types.User) error {
	return nil
}

func (m *mockStoreLogin) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockStoreLogin) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreLogin) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreLogin) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockStoreLogin) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}
