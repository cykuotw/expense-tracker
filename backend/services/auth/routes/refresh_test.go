package route

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/common"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type refreshStoreState struct {
	mu     sync.Mutex
	tokens map[string]types.RefreshToken
}

func newRefreshStoreState() *refreshStoreState {
	return &refreshStoreState{
		tokens: make(map[string]types.RefreshToken),
	}
}

func (s *refreshStoreState) CreateRefreshToken(token types.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token.FamilyID == uuid.Nil {
		token.FamilyID = token.ID
	}
	s.tokens[token.ID.String()] = token
	return nil
}

func (s *refreshStoreState) GetRefreshTokenByID(id string) (*types.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[id]
	if !ok {
		return nil, types.ErrInvalidToken
	}
	return &token, nil
}

func (s *refreshStoreState) RotateRefreshToken(id string, tokenHash string, successor types.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	predecessor, ok := s.tokens[id]
	if !ok || predecessor.TokenHash != tokenHash {
		return types.ErrInvalidToken
	}
	if predecessor.RevokedAt != nil {
		s.revokeFamilyLocked(predecessor.FamilyID)
		return types.ErrInvalidToken
	}
	if !time.Now().Before(predecessor.ExpiresAt) {
		return types.ErrInvalidToken
	}
	if successor.UserID != predecessor.UserID || successor.ID == uuid.Nil {
		return types.ErrInvalidToken
	}
	now := time.Now()
	predecessor.RevokedAt = &now
	s.tokens[id] = predecessor
	successor.FamilyID = predecessor.FamilyID
	s.tokens[successor.ID.String()] = successor
	return nil
}

func (s *refreshStoreState) RevokeRefreshTokenFamily(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	predecessor, ok := s.tokens[id]
	if !ok {
		return types.ErrInvalidToken
	}
	s.revokeFamilyLocked(predecessor.FamilyID)
	return nil
}

func (s *refreshStoreState) revokeFamilyLocked(familyID uuid.UUID) {
	now := time.Now()
	for tokenID, token := range s.tokens {
		if token.FamilyID != familyID || token.RevokedAt != nil {
			continue
		}
		token.RevokedAt = &now
		s.tokens[tokenID] = token
	}
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

	rotatedToken := responseCookieValue(t, rr, "refresh_token")
	rotatedClaims, err := auth.ParseTokenString(rotatedToken, "refresh")
	require.NoError(t, err)
	rotated, err := refreshStore.GetRefreshTokenByID(rotatedClaims.ID)
	require.NoError(t, err)
	assert.Equal(t, stored.FamilyID, rotated.FamilyID)
	assert.Nil(t, rotated.RevokedAt)
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
	familyID := uuid.New()
	if err := refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		FamilyID:  familyID,
		UserID:    userID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: now,
		RevokedAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	descendant := types.RefreshToken{
		ID:        uuid.New(),
		FamilyID:  familyID,
		UserID:    userID,
		TokenHash: "active-descendant",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	independent := types.RefreshToken{
		ID:        uuid.New(),
		FamilyID:  uuid.New(),
		UserID:    userID,
		TokenHash: "independent-session",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, refreshStore.CreateRefreshToken(descendant))
	require.NoError(t, refreshStore.CreateRefreshToken(independent))

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
	storedDescendant, err := refreshStore.GetRefreshTokenByID(descendant.ID.String())
	require.NoError(t, err)
	assert.NotNil(t, storedDescendant.RevokedAt)
	storedIndependent, err := refreshStore.GetRefreshTokenByID(independent.ID.String())
	require.NoError(t, err)
	assert.Nil(t, storedIndependent.RevokedAt)
}

func TestRefreshRevokedWithHashMismatchDoesNotRevokeFamily(t *testing.T) {
	userStore := loginUserStoreMock()
	refreshStore := newRefreshStoreState()
	handler := NewHandler(userStore, invitationStoreMock(), refreshStore)
	userID := uuid.New()
	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	require.NoError(t, err)
	now := time.Now()
	familyID := uuid.New()
	require.NoError(t, refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		FamilyID:  familyID,
		UserID:    userID,
		TokenHash: "different-hash",
		ExpiresAt: refreshExp,
		RevokedAt: &now,
		CreatedAt: now,
	}))
	descendant := types.RefreshToken{
		ID:        uuid.New(),
		FamilyID:  familyID,
		UserID:    userID,
		TokenHash: "still-active",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, refreshStore.CreateRefreshToken(descendant))

	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	response := httptest.NewRecorder()
	router := gin.New()
	router.POST("/auth/refresh", common.Make(handler.handleRefresh))
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	storedDescendant, err := refreshStore.GetRefreshTokenByID(descendant.ID.String())
	require.NoError(t, err)
	assert.Nil(t, storedDescendant.RevokedAt)
}

func TestRefreshAtomicClaimConflictReturnsUnauthorizedWithoutCookies(t *testing.T) {
	userID := uuid.New()
	refreshToken, refreshID, _, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	require.NoError(t, err)
	refreshStore := &baseRefreshStore{
		RotateRefreshTokenFn: func(id string, tokenHash string, successor types.RefreshToken) error {
			assert.Equal(t, refreshID, id)
			assert.Equal(t, auth.HashToken(refreshToken), tokenHash)
			assert.Equal(t, userID, successor.UserID)
			return types.ErrInvalidToken
		},
	}
	handler := NewHandler(loginUserStoreMock(), invitationStoreMock(), refreshStore)
	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	response := httptest.NewRecorder()
	router := gin.New()
	router.POST("/auth/refresh", common.Make(handler.handleRefresh))

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Empty(t, response.Result().Cookies())
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

func TestRefreshInactiveUser(t *testing.T) {
	userStore := &baseAuthUserStore{
		GetUserByIDFn: func(string) (*types.User, error) {
			return &types.User{IsActive: false}, nil
		},
	}
	refreshStore := newRefreshStoreState()
	handler := NewHandler(userStore, invitationStoreMock(), refreshStore)
	userID := uuid.New()
	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	assert.NoError(t, err)
	assert.NoError(t, refreshStore.CreateRefreshToken(types.RefreshToken{
		ID: uuid.MustParse(refreshID), UserID: userID, TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp, CreatedAt: time.Now(),
	}))
	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	response := httptest.NewRecorder()
	router := gin.New()
	router.POST("/auth/refresh", common.Make(handler.handleRefresh))

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusForbidden, response.Code)
	stored, err := refreshStore.GetRefreshTokenByID(refreshID)
	assert.NoError(t, err)
	assert.NotNil(t, stored.RevokedAt)
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
		FamilyID:  uuid.MustParse(refreshID),
		UserID:    userID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	related := types.RefreshToken{
		ID:        uuid.New(),
		FamilyID:  uuid.MustParse(refreshID),
		UserID:    userID,
		TokenHash: "related-session-token",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	independent := types.RefreshToken{
		ID:        uuid.New(),
		FamilyID:  uuid.New(),
		UserID:    userID,
		TokenHash: "independent-session-token",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, refreshStore.CreateRefreshToken(related))
	require.NoError(t, refreshStore.CreateRefreshToken(independent))

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
	storedRelated, err := refreshStore.GetRefreshTokenByID(related.ID.String())
	require.NoError(t, err)
	assert.NotNil(t, storedRelated.RevokedAt)
	storedIndependent, err := refreshStore.GetRefreshTokenByID(independent.ID.String())
	require.NoError(t, err)
	assert.Nil(t, storedIndependent.RevokedAt)
}

func responseCookieValue(t *testing.T, response *httptest.ResponseRecorder, name string) string {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	t.Fatalf("cookie %q was not set", name)
	return ""
}
