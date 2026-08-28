package route

import (
	"bytes"
	"context"
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
	handler := NewHandler(userStore, invitationStore, refreshStore, registrationStoreMock())

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

func TestRegisterInvitationErrorContract(t *testing.T) {
	tests := []struct {
		name       string
		storeError error
		status     int
		code       string
	}{
		{name: "invalid", storeError: types.ErrInvitationInvalid, status: http.StatusForbidden, code: "INVITATION_INVALID"},
		{name: "expired", storeError: types.ErrInvitationExpired, status: http.StatusForbidden, code: "INVITATION_EXPIRED"},
		{name: "used", storeError: types.ErrInvitationUsed, status: http.StatusForbidden, code: "INVITATION_USED"},
		{name: "email mismatch", storeError: types.ErrInvitationEmailMismatch, status: http.StatusForbidden, code: "INVITATION_EMAIL_MISMATCH"},
		{name: "account conflict", storeError: types.ErrAccountConflict, status: http.StatusConflict, code: "ACCOUNT_CONFLICT"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(registerUserStoreMock(), invitationStoreMock(), refreshStoreMock(), &baseRegistrationStore{
				CreateInvitedUserFn: func(ctx context.Context, token string, user types.User) error { return test.storeError },
			})
			payload := types.RegisterUserPayload{
				Firstname: "First", Lastname: "Last", Email: " User@Example.com ",
				Password: "longpassword", Token: "invite-token",
			}
			body, err := json.Marshal(payload)
			assert.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router := gin.New()
			router.POST("/register", handler.handleRegister)
			router.ServeHTTP(rr, req)

			assert.Equal(t, test.status, rr.Code)
			assert.Contains(t, rr.Body.String(), `"code":"`+test.code+`"`)
		})
	}
}

func TestRegisterNormalizesEmailBeforeAtomicCreation(t *testing.T) {
	var captured types.User
	handler := NewHandler(registerUserStoreMock(), invitationStoreMock(), refreshStoreMock(), &baseRegistrationStore{
		CreateInvitedUserFn: func(ctx context.Context, token string, user types.User) error {
			captured = user
			return nil
		},
	})
	payload := types.RegisterUserPayload{
		Firstname: "First", Lastname: "Last", Email: " User@Example.com ",
		Password: "longpassword", Token: "invite-token",
	}
	body, err := json.Marshal(payload)
	assert.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router := gin.New()
	router.POST("/register", handler.handleRegister)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, "user@example.com", captured.Email)
	assert.True(t, captured.HasLocalPassword)
}
