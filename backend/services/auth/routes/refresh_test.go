package route

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/backend/services/common"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type refreshStoreState struct {
	tokens map[string]types.RefreshToken
}

func newRefreshStoreState() *refreshStoreState {
	return &refreshStoreState{
		tokens: make(map[string]types.RefreshToken),
	}
}

func (s *refreshStoreState) CreateRefreshToken(token types.RefreshToken) error {
	s.tokens[token.ID.String()] = token
	return nil
}

func (s *refreshStoreState) GetRefreshTokenByID(id string) (*types.RefreshToken, error) {
	token, ok := s.tokens[id]
	if !ok {
		return nil, types.ErrInvalidToken
	}
	return &token, nil
}

func (s *refreshStoreState) RevokeRefreshToken(id string) error {
	token, ok := s.tokens[id]
	if !ok {
		return types.ErrInvalidToken
	}
	now := time.Now()
	token.RevokedAt = &now
	s.tokens[id] = token
	return nil
}

func TestRefreshSuccess(t *testing.T) {
	userStore := loginUserStoreMock()
	invitationStore := invitationStoreMock()
	refreshStore := newRefreshStoreState()
	handler := NewHandler(userStore, invitationStore, refreshStore)

	userID := uuid.New()
	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		UserID:    userID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})

	rr := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/auth/refresh", common.Make(handler.handleRefresh))
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	stored, err := refreshStore.GetRefreshTokenByID(refreshID)
	assert.NoError(t, err)
	assert.NotNil(t, stored.RevokedAt)
}

func TestRefreshRevoked(t *testing.T) {
	userStore := loginUserStoreMock()
	invitationStore := invitationStoreMock()
	refreshStore := newRefreshStoreState()
	handler := NewHandler(userStore, invitationStore, refreshStore)

	userID := uuid.New()
	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		UserID:    userID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: now,
		RevokedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})

	rr := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/auth/refresh", common.Make(handler.handleRefresh))
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRefreshExpired(t *testing.T) {
	userStore := loginUserStoreMock()
	invitationStore := invitationStoreMock()
	refreshStore := newRefreshStoreState()
	handler := NewHandler(userStore, invitationStore, refreshStore)

	originalExp := config.Envs.RefreshJWTExpirationInSeconds
	config.Envs.RefreshJWTExpirationInSeconds = -1
	t.Cleanup(func() {
		config.Envs.RefreshJWTExpirationInSeconds = originalExp
	})

	userID := uuid.New()
	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		UserID:    userID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})

	rr := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/auth/refresh", common.Make(handler.handleRefresh))
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	userStore := loginUserStoreMock()
	invitationStore := invitationStoreMock()
	refreshStore := newRefreshStoreState()
	handler := NewHandler(userStore, invitationStore, refreshStore)

	userID := uuid.New()
	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		UserID:    userID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})

	rr := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/logout", common.Make(handler.handleLogout))
	router.ServeHTTP(rr, req)

	stored, err := refreshStore.GetRefreshTokenByID(refreshID)
	assert.NoError(t, err)
	assert.NotNil(t, stored.RevokedAt)
}
