package route

import (
	"crypto/rand"
	"encoding/base64"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
)

func initThirdParty() {
	goth.UseProviders(
		google.New(config.Envs.GoogleClientId,
			config.Envs.GoogleClientSecret,
			config.Envs.GoogleCallbackUrl,
			"profile", "email"),
	)
}

func (h *Handler) handleThirdParty(c *gin.Context) error {
	provider := c.Param("provider")
	gothProvider, err := goth.GetProvider(provider)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return err
	}

	state, err := generateOAuthState()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	session, err := gothProvider.BeginAuth(state)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	storeOAuthSession(c, provider, state, session.Marshal())

	authURL, err := session.GetAuthURL()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
	return nil
}

func (h *Handler) handleThirdPartyCallback(c *gin.Context) error {
	provider := c.Param("provider")
	gothProvider, err := goth.GetProvider(provider)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return err
	}

	stateCookie, sessionCookie, err := loadOAuthSession(c, provider)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return err
	}

	if c.Query("state") == "" || c.Query("state") != stateCookie {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidCSRFToken)
		return types.ErrInvalidCSRFToken
	}

	session, err := gothProvider.UnmarshalSession(sessionCookie)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return err
	}

	if _, err := session.Authorize(gothProvider, c.Request.URL.Query()); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	gothUser, err := gothProvider.FetchUser(session)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	clearOAuthSession(c, provider)

	exist, err := h.store.CheckEmailExist(gothUser.Email)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}

	var user *types.User
	if !exist {
		// register
		hashedPassword, err := auth.HashPassword(gothUser.UserID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return err
		}
		user = &types.User{
			ID:             uuid.New(),
			Username:       gothUser.NickName,
			Nickname:       gothUser.NickName,
			Firstname:      gothUser.FirstName,
			Lastname:       gothUser.LastName,
			Email:          gothUser.Email,
			PasswordHashed: hashedPassword,
			ExternalType:   provider,
			ExternalID:     gothUser.UserID,
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
		user, err = h.store.GetUserByEmail(gothUser.Email)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return err
		}
		if !auth.ValidatePassword(user.PasswordHashed, gothUser.UserID) {
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

func storeOAuthSession(c *gin.Context, provider string, state string, session string) {
	http.SetCookie(c.Writer, buildOAuthCookie(oauthStateCookieName(provider), state, int(config.Envs.ThirdPartySessionMaxAge), c))
	http.SetCookie(c.Writer, buildOAuthCookie(oauthSessionCookieName(provider), encodeOAuthSession(session), int(config.Envs.ThirdPartySessionMaxAge), c))
}

func loadOAuthSession(c *gin.Context, provider string) (string, string, error) {
	state, err := c.Cookie(oauthStateCookieName(provider))
	if err != nil {
		return "", "", fmt.Errorf("missing oauth state")
	}

	encodedSession, err := c.Cookie(oauthSessionCookieName(provider))
	if err != nil {
		return "", "", fmt.Errorf("missing oauth session")
	}

	session, err := decodeOAuthSession(encodedSession)
	if err != nil {
		return "", "", err
	}

	return state, session, nil
}

func clearOAuthSession(c *gin.Context, provider string) {
	http.SetCookie(c.Writer, buildOAuthCookie(oauthStateCookieName(provider), "", -1, c))
	http.SetCookie(c.Writer, buildOAuthCookie(oauthSessionCookieName(provider), "", -1, c))
}

func oauthStateCookieName(provider string) string {
	return fmt.Sprintf("oauth_%s_state", provider)
}

func oauthSessionCookieName(provider string) string {
	return fmt.Sprintf("oauth_%s_session", provider)
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

func encodeOAuthSession(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeOAuthSession(value string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
