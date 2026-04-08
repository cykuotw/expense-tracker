package route

import (
	"context"
	"encoding/json"
	"expense-tracker/backend/config"
	googleAuth "expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/services/common"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockGoogleService struct {
	resolveUserFromClaimsFn func(claims *types.VerifiedGoogleClaims) (*types.User, error)
}

func (s *mockGoogleService) ResolveUserFromClaims(claims *types.VerifiedGoogleClaims) (*types.User, error) {
	if s.resolveUserFromClaimsFn != nil {
		return s.resolveUserFromClaimsFn(claims)
	}
	return nil, nil
}

type mockGoogleVerifier struct {
	verifyFn func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error)
}

func (v *mockGoogleVerifier) VerifyGoogleIDToken(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
	if v.verifyFn != nil {
		return v.verifyFn(ctx, rawToken)
	}
	return nil, nil
}

func TestRegisterRoutesGoogleExchangeGating(t *testing.T) {
	originalEnabled := config.Envs.GoogleOAuthEnabled
	originalClientID := config.Envs.GoogleClientId
	originalExchangeMode := config.Envs.GoogleExchangeMode
	defer func() {
		config.Envs.GoogleOAuthEnabled = originalEnabled
		config.Envs.GoogleClientId = originalClientID
		config.Envs.GoogleExchangeMode = originalExchangeMode
	}()

	t.Run("inprocess mode enabled", func(t *testing.T) {
		config.Envs.GoogleOAuthEnabled = true
		config.Envs.GoogleClientId = "test-google-client-id"
		config.Envs.GoogleExchangeMode = config.GoogleExchangeInProcess

		router := gin.New()
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.RegisterRoutes(router.Group(""))

		foundExchange := false
		for _, route := range router.Routes() {
			if route.Method == http.MethodPost && route.Path == "/auth/google/exchange" {
				foundExchange = true
			}
		}

		assert.True(t, foundExchange)
	})

	t.Run("missing google oauth config does not register exchange route", func(t *testing.T) {
		config.Envs.GoogleOAuthEnabled = false
		config.Envs.GoogleClientId = ""
		config.Envs.GoogleExchangeMode = config.GoogleExchangeInProcess

		router := gin.New()
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.RegisterRoutes(router.Group(""))

		for _, route := range router.Routes() {
			assert.False(t, route.Method == http.MethodPost && route.Path == "/auth/google/exchange")
		}
	})

	t.Run("enabled google oauth always registers exchange route", func(t *testing.T) {
		config.Envs.GoogleOAuthEnabled = true
		config.Envs.GoogleClientId = "test-google-client-id"
		config.Envs.GoogleExchangeMode = config.GoogleExchangeInProcess

		router := gin.New()
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.RegisterRoutes(router.Group(""))

		foundExchange := false
		for _, route := range router.Routes() {
			if route.Method == http.MethodPost && route.Path == "/auth/google/exchange" {
				foundExchange = true
			}
		}

		assert.True(t, foundExchange)
	})

	t.Run("upstream verified mode also registers exchange route", func(t *testing.T) {
		config.Envs.GoogleOAuthEnabled = true
		config.Envs.GoogleClientId = "test-google-client-id"
		config.Envs.GoogleExchangeMode = config.GoogleExchangeUpstreamVerified

		router := gin.New()
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.RegisterRoutes(router.Group(""))

		foundExchange := false
		for _, route := range router.Routes() {
			if route.Method == http.MethodPost && route.Path == "/auth/google/exchange" {
				foundExchange = true
			}
		}
		assert.True(t, foundExchange)
	})
}

func TestHandleGoogleExchangeInProcess(t *testing.T) {
	originalEnabled := config.Envs.GoogleOAuthEnabled
	originalClientID := config.Envs.GoogleClientId
	originalExchangeMode := config.Envs.GoogleExchangeMode
	defer func() {
		config.Envs.GoogleOAuthEnabled = originalEnabled
		config.Envs.GoogleClientId = originalClientID
		config.Envs.GoogleExchangeMode = originalExchangeMode
	}()

	config.Envs.GoogleOAuthEnabled = true
	config.Envs.GoogleClientId = "test-google-client-id"
	config.Envs.GoogleExchangeMode = config.GoogleExchangeInProcess
	gin.SetMode(gin.ReleaseMode)

	t.Run("missing authorization header", func(t *testing.T) {
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.googleVerifier = &mockGoogleVerifier{
			verifyFn: func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
				t.Fatalf("verifier should not be called")
				return nil, nil
			},
		}
		handler.googleService = &mockGoogleService{
			resolveUserFromClaimsFn: func(claims *types.VerifiedGoogleClaims) (*types.User, error) {
				t.Fatalf("resolver should not be called")
				return nil, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", nil)
		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/auth/google/exchange", common.Make(handler.handleGoogleExchangeInProcess))

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		var response struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
		assert.Equal(t, "missing authorization header", response.Error)
		assert.Equal(t, "missing_authorization_header", response.Code)
	})

	t.Run("successful exchange issues auth cookies", func(t *testing.T) {
		userID := uuid.New()
		refreshCreated := false
		refreshStore := &baseRefreshStore{
			CreateRefreshTokenFn: func(token types.RefreshToken) error {
				refreshCreated = true
				assert.Equal(t, userID, token.UserID)
				return nil
			},
		}
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStore)
		handler.googleVerifier = &mockGoogleVerifier{
			verifyFn: func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
				assert.Equal(t, "test-id-token", rawToken)
				return &types.VerifiedGoogleClaims{
					Subject:       "google-sub-123",
					Email:         "user@example.com",
					EmailVerified: boolPtr(true),
				}, nil
			},
		}
		handler.googleService = &mockGoogleService{
			resolveUserFromClaimsFn: func(claims *types.VerifiedGoogleClaims) (*types.User, error) {
				assert.Equal(t, "google-sub-123", claims.Subject)
				return &types.User{
					ID:   userID,
					Role: "user",
				}, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", nil)
		req.Header.Set("Authorization", "Bearer test-id-token")
		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/auth/google/exchange", common.Make(handler.handleGoogleExchangeInProcess))

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.True(t, refreshCreated)
		cookies := rr.Result().Cookies()
		assert.Len(t, cookies, 2)
		assert.Equal(t, "access_token", cookies[0].Name)
		assert.Equal(t, "refresh_token", cookies[1].Name)
	})

	t.Run("email collision returns conflict", func(t *testing.T) {
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.googleVerifier = &mockGoogleVerifier{
			verifyFn: func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
				return &types.VerifiedGoogleClaims{
					Subject:       "google-sub-123",
					Email:         "user@example.com",
					EmailVerified: boolPtr(true),
				}, nil
			},
		}
		handler.googleService = &mockGoogleService{
			resolveUserFromClaimsFn: func(claims *types.VerifiedGoogleClaims) (*types.User, error) {
				return nil, types.ErrGoogleAccountConflict
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", nil)
		req.Header.Set("Authorization", "Bearer test-id-token")
		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/auth/google/exchange", common.Make(handler.handleGoogleExchangeInProcess))

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)

		var response struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
		assert.Equal(t, "google_account_conflict", response.Code)
	})

	t.Run("unverified email blocks account creation", func(t *testing.T) {
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.googleVerifier = &mockGoogleVerifier{
			verifyFn: func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
				return &types.VerifiedGoogleClaims{
					Subject:       "google-sub-123",
					Email:         "user@example.com",
					EmailVerified: boolPtr(true),
				}, nil
			},
		}
		handler.googleService = &mockGoogleService{
			resolveUserFromClaimsFn: func(claims *types.VerifiedGoogleClaims) (*types.User, error) {
				return nil, types.ErrGoogleEmailNotVerified
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", nil)
		req.Header.Set("Authorization", "Bearer test-id-token")
		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/auth/google/exchange", common.Make(handler.handleGoogleExchangeInProcess))

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var response struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
		assert.Equal(t, "google_email_not_verified", response.Code)
	})

	t.Run("invalid google token returns unauthorized", func(t *testing.T) {
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.googleVerifier = &mockGoogleVerifier{
			verifyFn: func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
				return nil, types.ErrInvalidGoogleIDToken
			},
		}
		handler.googleService = &mockGoogleService{
			resolveUserFromClaimsFn: func(claims *types.VerifiedGoogleClaims) (*types.User, error) {
				t.Fatalf("resolver should not be called")
				return nil, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", nil)
		req.Header.Set("Authorization", "Bearer test-id-token")
		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/auth/google/exchange", common.Make(handler.handleGoogleExchangeInProcess))

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestHandleGoogleExchangeUpstreamVerified(t *testing.T) {
	originalEnabled := config.Envs.GoogleOAuthEnabled
	originalClientID := config.Envs.GoogleClientId
	originalExchangeMode := config.Envs.GoogleExchangeMode
	defer func() {
		config.Envs.GoogleOAuthEnabled = originalEnabled
		config.Envs.GoogleClientId = originalClientID
		config.Envs.GoogleExchangeMode = originalExchangeMode
	}()

	config.Envs.GoogleOAuthEnabled = true
	config.Envs.GoogleClientId = "test-google-client-id"
	config.Envs.GoogleExchangeMode = config.GoogleExchangeUpstreamVerified
	gin.SetMode(gin.ReleaseMode)

	t.Run("upstream verified mode resolves user from verified claims in context", func(t *testing.T) {
		userID := uuid.New()
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.googleVerifier = &mockGoogleVerifier{
			verifyFn: func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
				t.Fatalf("inprocess verifier should not be called in upstream verified mode")
				return nil, nil
			},
		}
		handler.googleService = &mockGoogleService{
			resolveUserFromClaimsFn: func(claims *types.VerifiedGoogleClaims) (*types.User, error) {
				assert.Equal(t, "google-sub-123", claims.Subject)
				return &types.User{
					ID:   userID,
					Role: "user",
				}, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", nil)
		req = req.WithContext(googleAuth.ContextWithVerifiedClaims(req.Context(), &types.VerifiedGoogleClaims{
			Subject:       "google-sub-123",
			Email:         "user@example.com",
			EmailVerified: boolPtr(true),
		}))
		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/auth/google/exchange", common.Make(handler.handleGoogleExchangeUpstreamVerified))

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("upstream verified mode requires verified claims in context", func(t *testing.T) {
		handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStoreMock())
		handler.googleVerifier = &mockGoogleVerifier{
			verifyFn: func(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
				t.Fatalf("inprocess verifier should not be called in upstream verified mode")
				return nil, nil
			},
		}
		handler.googleService = &mockGoogleService{
			resolveUserFromClaimsFn: func(claims *types.VerifiedGoogleClaims) (*types.User, error) {
				t.Fatalf("resolver should not be called when claims are missing")
				return nil, nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/google/exchange", nil)
		rr := httptest.NewRecorder()
		router := gin.New()
		router.POST("/auth/google/exchange", common.Make(handler.handleGoogleExchangeUpstreamVerified))

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		var response struct {
			Code string `json:"code"`
		}
		assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
		assert.Equal(t, "google_claims_unavailable", response.Code)
	})
}

func boolPtr(value bool) *bool {
	return &value
}
