package route

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

var (
	googleOAuthEndpoint = googleoauth.Endpoint
	googleUserInfoURL   = "https://openidconnect.googleapis.com/v1/userinfo"
	exchangeGoogleCode  = func(ctx context.Context, code string) (*oauth2.Token, error) {
		return googleOAuthConfig().Exchange(ctx, code)
	}
	loadGoogleUser = fetchGoogleUser
)

type googleUserInfo struct {
	Subject    string `json:"sub"`
	Email      string `json:"email"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Name       string `json:"name"`
}

func (h *Handler) handleThirdParty(c *gin.Context) error {
	provider := c.Param("provider")
	if err := validateOAuthProvider(provider); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return err
	}

	state, err := generateOAuthState()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	storeOAuthState(c, provider, state)
	authURL := googleOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)

	c.Redirect(http.StatusTemporaryRedirect, authURL)
	return nil
}

func (h *Handler) handleThirdPartyCallback(c *gin.Context) error {
	provider := c.Param("provider")
	if err := validateOAuthProvider(provider); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return err
	}

	stateCookie, err := loadOAuthState(c, provider)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return err
	}

	if c.Query("state") == "" || c.Query("state") != stateCookie {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidCSRFToken)
		return types.ErrInvalidCSRFToken
	}

	token, err := exchangeGoogleCode(c.Request.Context(), c.Query("code"))
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	googleUser, err := loadGoogleUser(c.Request.Context(), token)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	clearOAuthState(c, provider)

	exist, err := h.store.CheckEmailExist(googleUser.Email)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	var user *types.User
	if !exist {
		// register
		hashedPassword, err := auth.HashPassword(googleUser.Subject)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return err
		}
		user = &types.User{
			ID:             uuid.New(),
			Username:       googleUserNickname(googleUser),
			Nickname:       googleUserNickname(googleUser),
			Firstname:      googleUser.GivenName,
			Lastname:       googleUser.FamilyName,
			Email:          googleUser.Email,
			PasswordHashed: hashedPassword,
			ExternalType:   provider,
			ExternalID:     googleUser.Subject,
			CreateTime:     time.Now(),
			IsActive:       true,
			Role:           "user",
		}
		err = h.store.CreateUser(*user)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return err
		}
	} else {
		// login
		user, err = h.store.GetUserByEmail(googleUser.Email)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return err
		}
		if !auth.ValidatePassword(user.PasswordHashed, googleUser.Subject) {
			utils.WriteError(c, http.StatusBadRequest, types.ErrPasswordNotMatch)
			return types.ErrPasswordNotMatch
		}
	}

	secret := []byte(config.Envs.JWTSecret)
	accessToken, err := auth.CreateJWT(secret, user.ID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	refreshToken, refreshID, refreshExp, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), user.ID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}
	if err := h.refreshStore.CreateRefreshToken(types.RefreshToken{
		ID:        uuid.MustParse(refreshID),
		UserID:    user.ID,
		TokenHash: auth.HashToken(refreshToken),
		ExpiresAt: refreshExp,
		CreatedAt: time.Now(),
	}); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	setAuthCookies(c, accessToken, refreshToken)

	c.Redirect(http.StatusTemporaryRedirect, config.Envs.FrontendOrigin)

	return nil
}

func validateOAuthProvider(provider string) error {
	if provider != "google" {
		return fmt.Errorf("unsupported oauth provider: %s", provider)
	}
	return nil
}

func googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.Envs.GoogleClientId,
		ClientSecret: config.Envs.GoogleClientSecret,
		RedirectURL:  config.Envs.GoogleCallbackUrl,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     googleOAuthEndpoint,
	}
}

func fetchGoogleUser(ctx context.Context, token *oauth2.Token) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo request failed: %s", resp.Status)
	}

	var user googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	if user.Subject == "" || user.Email == "" {
		return nil, fmt.Errorf("google userinfo response missing required fields")
	}

	return &user, nil
}

func generateOAuthState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (h *Handler) handleAuthMe(c *gin.Context) error {

	userId, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return err
	}

	user, err := h.store.GetUserByID(userId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	utils.WriteJSON(c, http.StatusOK, gin.H{"userID": userId, "role": user.Role})

	return nil
}

func storeOAuthState(c *gin.Context, provider string, state string) {
	http.SetCookie(c.Writer, buildOAuthCookie(oauthStateCookieName(provider), state, int(config.Envs.ThirdPartySessionMaxAge), c))
}

func loadOAuthState(c *gin.Context, provider string) (string, error) {
	state, err := c.Cookie(oauthStateCookieName(provider))
	if err != nil {
		return "", fmt.Errorf("missing oauth state")
	}
	return state, nil
}

func clearOAuthState(c *gin.Context, provider string) {
	http.SetCookie(c.Writer, buildOAuthCookie(oauthStateCookieName(provider), "", -1, c))
}

func oauthStateCookieName(provider string) string {
	return fmt.Sprintf("oauth_%s_state", provider)
}

func buildOAuthCookie(name string, value string, maxAge int, c *gin.Context) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   config.Envs.AuthCookieDomain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   resolveAuthCookieSecure(c),
		SameSite: config.Envs.AuthCookieSameSite,
	}
}

func googleUserNickname(user *googleUserInfo) string {
	if user.GivenName != "" {
		return user.GivenName
	}
	if user.Name != "" {
		return user.Name
	}
	if localPart, _, found := strings.Cut(user.Email, "@"); found && localPart != "" {
		return sanitizeUsername(localPart)
	}
	return "google-user"
}

func sanitizeUsername(value string) string {
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, value)
	sanitized = strings.Trim(sanitized, "-_")
	if sanitized == "" {
		return "google-user"
	}
	if len(sanitized) > 32 {
		return sanitized[:32]
	}
	return sanitized
}
