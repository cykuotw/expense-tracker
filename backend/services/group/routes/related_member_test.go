package group

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetRelatedMemberReturnsEmptyArray(t *testing.T) {
	store := groupStoreMock()
	store.CheckGroupUserPairExistFn = func(groupID string, userID string) (bool, error) {
		return groupID == mockGroupId.String() && userID == mockUserId.String(), nil
	}
	store.GetRelatedUserFn = func(string, string) ([]*types.RelatedMember, error) {
		return nil, nil
	}

	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), mockUserId)
	assert.NoError(t, err)

	request := httptest.NewRequest(
		http.MethodGet,
		"/related_member?g="+mockGroupId.String(),
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router := gin.New()
	router.GET("/related_member", NewHandler(store, userStoreMock()).handleGetRelatedMember)

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, "[]", response.Body.String())
}
