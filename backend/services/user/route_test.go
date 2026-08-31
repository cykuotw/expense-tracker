package user

import (
	"bytes"
	"encoding/json"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type userLookupStore struct {
	user *types.User
}

func (s userLookupStore) GetUserByEmail(string) (*types.User, error) {
	return s.user, nil
}

func (userLookupStore) GetUserByExternalIdentity(string, string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}

func (userLookupStore) GetUserByID(string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}

func (userLookupStore) GetUsernameByID(string) (string, error) {
	return "", types.ErrUserNotExist
}

func (userLookupStore) CreateUser(types.User) error { return nil }

func (userLookupStore) CheckEmailExist(string) (bool, error) { return false, nil }

func (userLookupStore) CheckUserExistByEmail(string) (bool, error) { return false, nil }

func (userLookupStore) CheckUserExistByID(string) (bool, error) { return false, nil }

func (userLookupStore) CheckUserExistByUsername(string) (bool, error) { return false, nil }

func TestHandleGetUserInfoByEmailReturnsOnlyPublicIdentity(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	userID := uuid.New()
	handler := NewHandler(userLookupStore{user: &types.User{
		ID:             userID,
		Username:       "member",
		Email:          "member@example.com",
		PasswordHashed: "must-not-be-returned",
		ExternalID:     "private-identity",
	}})
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/userInfo", bytes.NewBufferString(`{"email":"member@example.com"}`))

	require.NoError(t, handler.handleGetUserInfoByEmail(context))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	require.Len(t, response, 2)
	require.JSONEq(t, `"`+userID.String()+`"`, string(response["id"]))
	require.JSONEq(t, `"member"`, string(response["username"]))
	_, hasPasswordHash := response["passwordHashed"]
	_, hasExternalID := response["externalId"]
	require.False(t, hasPasswordHash)
	require.False(t, hasExternalID)
}
