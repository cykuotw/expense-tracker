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
	"github.com/stretchr/testify/require"
)

func TestInvitationExchangeUsesBodyAndIssuesOnlyShortLivedSessionCookie(t *testing.T) {
	var receivedToken string
	var receivedSession string
	handler := NewHandler(registerUserStoreMock(), &baseInvitationStore{
		ExchangeInvitationFn: func(token string, registrationSession string) (*types.Invitation, error) {
			receivedToken = token
			receivedSession = registrationSession
			return &types.Invitation{Email: "invited@example.test"}, nil
		},
	}, refreshStoreMock())

	body := bytes.NewBufferString(`{"token":"fragment-secret"}`)
	request := httptest.NewRequest(http.MethodPost, "/register/invitation/exchange", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router := gin.New()
	router.POST("/register/invitation/exchange", handler.handleInvitationExchange)
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "fragment-secret", receivedToken)
	assert.NotEmpty(t, receivedSession)
	assert.NotEqual(t, receivedToken, receivedSession)

	var payload types.InvitationResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	assert.True(t, payload.Valid)
	assert.Equal(t, "invited@example.test", payload.Email)

	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, registrationSessionCookieName, cookies[0].Name)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, int(registrationSessionTTL.Seconds()), cookies[0].MaxAge)
}

func TestInvitationExchangeRejectsMissingSecret(t *testing.T) {
	handler := NewHandler(registerUserStoreMock(), invitationStoreMock(), refreshStoreMock())
	request := httptest.NewRequest(http.MethodPost, "/register/invitation/exchange", bytes.NewBufferString(`{}`))
	response := httptest.NewRecorder()
	router := gin.New()
	router.POST("/register/invitation/exchange", handler.handleInvitationExchange)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, response.Result().Cookies())
}
