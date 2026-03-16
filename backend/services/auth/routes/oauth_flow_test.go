package route

import (
	"context"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestHandleThirdPartyRedirectsToGoogleAuthURL(t *testing.T) {
	originalClientID := config.Envs.GoogleClientId
	originalCallbackURL := config.Envs.GoogleCallbackUrl
	originalEndpoint := googleOAuthEndpoint
	config.Envs.GoogleClientId = "client-id"
	config.Envs.GoogleCallbackUrl = "http://localhost:8080/api/v0/auth/google/callback"
	googleOAuthEndpoint = oauth2.Endpoint{
		AuthURL:  "https://accounts.example.test/o/oauth2/auth",
		TokenURL: "https://accounts.example.test/token",
	}
	t.Cleanup(func() {
		config.Envs.GoogleClientId = originalClientID
		config.Envs.GoogleCallbackUrl = originalCallbackURL
		googleOAuthEndpoint = originalEndpoint
	})

	gin.SetMode(gin.ReleaseMode)
	handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
	router := gin.New()
	router.POST("/auth/:provider", func(c *gin.Context) {
		_ = handler.handleThirdParty(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/google", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	location := rr.Header().Get("Location")
	assert.Contains(t, location, "https://accounts.example.test/o/oauth2/auth")
	assert.Contains(t, location, "client_id=client-id")
	assert.Contains(t, location, url.QueryEscape(config.Envs.GoogleCallbackUrl))
	assert.NotEmpty(t, rr.Result().Cookies())
	assert.Equal(t, oauthStateCookieName("google"), rr.Result().Cookies()[0].Name)
}

func TestHandleThirdPartyCallbackCreatesUserFromGoogleProfile(t *testing.T) {
	originalClientID := config.Envs.GoogleClientId
	originalClientSecret := config.Envs.GoogleClientSecret
	originalCallbackURL := config.Envs.GoogleCallbackUrl
	originalFrontendOrigin := config.Envs.FrontendOrigin
	originalJWTSecret := config.Envs.JWTSecret
	originalRefreshSecret := config.Envs.RefreshJWTSecret
	originalEndpoint := googleOAuthEndpoint
	originalUserInfoURL := googleUserInfoURL
	originalExchange := exchangeGoogleCode
	originalLoadUser := loadGoogleUser
	config.Envs.GoogleClientId = "client-id"
	config.Envs.GoogleClientSecret = "client-secret"
	config.Envs.GoogleCallbackUrl = "http://localhost:8080/auth/google/callback"
	config.Envs.FrontendOrigin = "http://localhost:5173"
	config.Envs.JWTSecret = "jwt-secret"
	config.Envs.RefreshJWTSecret = "refresh-secret"
	t.Cleanup(func() {
		config.Envs.GoogleClientId = originalClientID
		config.Envs.GoogleClientSecret = originalClientSecret
		config.Envs.GoogleCallbackUrl = originalCallbackURL
		config.Envs.FrontendOrigin = originalFrontendOrigin
		config.Envs.JWTSecret = originalJWTSecret
		config.Envs.RefreshJWTSecret = originalRefreshSecret
		googleOAuthEndpoint = originalEndpoint
		googleUserInfoURL = originalUserInfoURL
		exchangeGoogleCode = originalExchange
		loadGoogleUser = originalLoadUser
	})

	var createdUser types.User
	userStore := &baseAuthUserStore{
		CheckEmailExistFn: func(email string) (bool, error) {
			assert.Equal(t, "user@example.com", email)
			return false, nil
		},
		CreateUserFn: func(user types.User) error {
			createdUser = user
			return nil
		},
	}
	refreshStore := &baseRefreshStore{
		CreateRefreshTokenFn: func(token types.RefreshToken) error {
			assert.Equal(t, createdUser.ID, token.UserID)
			return nil
		},
	}

	exchangeGoogleCode = func(ctx context.Context, code string) (*oauth2.Token, error) {
		assert.Equal(t, "test-code", code)
		return &oauth2.Token{AccessToken: "google-access-token"}, nil
	}
	loadGoogleUser = func(ctx context.Context, token *oauth2.Token) (*googleUserInfo, error) {
		assert.Equal(t, "google-access-token", token.AccessToken)
		return &googleUserInfo{
			Subject:    "google-user-123",
			Email:      "user@example.com",
			GivenName:  "Taylor",
			FamilyName: "Swift",
			Name:       "Taylor Swift",
		}, nil
	}

	gin.SetMode(gin.ReleaseMode)
	handler := NewHandler(userStore, invitationStoreMock(), refreshStore)
	router := gin.New()
	router.GET("/auth/:provider/callback", func(c *gin.Context) {
		_ = handler.handleThirdPartyCallback(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=test-state&code=test-code", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName("google"), Value: "test-state"})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	assert.Equal(t, config.Envs.FrontendOrigin, rr.Header().Get("Location"))
	assert.Equal(t, "user@example.com", createdUser.Email)
	assert.Equal(t, "Taylor", createdUser.Username)
	assert.Equal(t, "google", createdUser.ExternalType)
	assert.Equal(t, "google-user-123", createdUser.ExternalID)

	cookies := rr.Result().Cookies()
	assert.GreaterOrEqual(t, len(cookies), 3)
}

func TestHandleThirdPartyCallbackRejectsInvalidState(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
	router := gin.New()
	router.GET("/auth/:provider/callback", func(c *gin.Context) {
		_ = handler.handleThirdPartyCallback(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=bad-state&code=test-code", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName("google"), Value: "expected-state"})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGoogleUserNicknameFallbacks(t *testing.T) {
	assert.Equal(t, "Given", googleUserNickname(&googleUserInfo{
		GivenName: "Given",
		Email:     "user@example.com",
	}))
	assert.Equal(t, "Display Name", googleUserNickname(&googleUserInfo{
		Name:  "Display Name",
		Email: "user@example.com",
	}))
	assert.Equal(t, "local-part", googleUserNickname(&googleUserInfo{
		Email: "local.part@example.com",
	}))
	assert.Equal(t, "google-user", googleUserNickname(&googleUserInfo{}))
}

func TestSanitizeUsername(t *testing.T) {
	assert.Equal(t, "hello-world", sanitizeUsername("Hello World"))
	assert.Equal(t, "abc_123", sanitizeUsername("abc_123"))
	assert.Equal(t, "google-user", sanitizeUsername("!!!"))
	assert.Equal(t, strings.Repeat("a", 32), sanitizeUsername(strings.Repeat("a", 40)))
}

func TestHandleThirdPartyCallbackExistingUserLogin(t *testing.T) {
	originalClientID := config.Envs.GoogleClientId
	originalClientSecret := config.Envs.GoogleClientSecret
	originalCallbackURL := config.Envs.GoogleCallbackUrl
	originalFrontendOrigin := config.Envs.FrontendOrigin
	originalJWTSecret := config.Envs.JWTSecret
	originalRefreshSecret := config.Envs.RefreshJWTSecret
	originalEndpoint := googleOAuthEndpoint
	originalUserInfoURL := googleUserInfoURL
	originalExchange := exchangeGoogleCode
	originalLoadUser := loadGoogleUser
	config.Envs.GoogleClientId = "client-id"
	config.Envs.GoogleClientSecret = "client-secret"
	config.Envs.GoogleCallbackUrl = "http://localhost:8080/auth/google/callback"
	config.Envs.FrontendOrigin = "http://localhost:5173"
	config.Envs.JWTSecret = "jwt-secret"
	config.Envs.RefreshJWTSecret = "refresh-secret"
	t.Cleanup(func() {
		config.Envs.GoogleClientId = originalClientID
		config.Envs.GoogleClientSecret = originalClientSecret
		config.Envs.GoogleCallbackUrl = originalCallbackURL
		config.Envs.FrontendOrigin = originalFrontendOrigin
		config.Envs.JWTSecret = originalJWTSecret
		config.Envs.RefreshJWTSecret = originalRefreshSecret
		googleOAuthEndpoint = originalEndpoint
		googleUserInfoURL = originalUserInfoURL
		exchangeGoogleCode = originalExchange
		loadGoogleUser = originalLoadUser
	})

	hashedPassword, _ := auth.HashPassword("google-user-123")
	existingID := uuid.New()
	userStore := &baseAuthUserStore{
		CheckEmailExistFn: func(email string) (bool, error) { return true, nil },
		GetUserByEmailFn: func(email string) (*types.User, error) {
			return &types.User{
				ID:             existingID,
				Email:          email,
				PasswordHashed: hashedPassword,
				CreateTime:     time.Now(),
			}, nil
		},
	}

	exchangeGoogleCode = func(ctx context.Context, code string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "google-access-token"}, nil
	}
	loadGoogleUser = func(ctx context.Context, token *oauth2.Token) (*googleUserInfo, error) {
		return &googleUserInfo{
			Subject: "google-user-123",
			Email:   "user@example.com",
		}, nil
	}

	handler := NewHandler(userStore, invitationStoreMock(), refreshStoreMock())
	router := gin.New()
	router.GET("/auth/:provider/callback", func(c *gin.Context) {
		_ = handler.handleThirdPartyCallback(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=test-state&code=test-code", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName("google"), Value: "test-state"})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	assert.Equal(t, config.Envs.FrontendOrigin, rr.Header().Get("Location"))
}

func TestHandleThirdPartyCallbackFailsOnGoogleExchangeError(t *testing.T) {
	originalExchange := exchangeGoogleCode
	originalLoadUser := loadGoogleUser
	t.Cleanup(func() {
		exchangeGoogleCode = originalExchange
		loadGoogleUser = originalLoadUser
	})

	exchangeGoogleCode = func(ctx context.Context, code string) (*oauth2.Token, error) {
		return nil, fmt.Errorf("exchange failed")
	}
	loadGoogleUser = func(ctx context.Context, token *oauth2.Token) (*googleUserInfo, error) {
		t.Fatal("loadGoogleUser should not be called")
		return nil, nil
	}

	gin.SetMode(gin.ReleaseMode)
	handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
	router := gin.New()
	router.GET("/auth/:provider/callback", func(c *gin.Context) {
		_ = handler.handleThirdPartyCallback(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=test-state&code=test-code", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName("google"), Value: "test-state"})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
