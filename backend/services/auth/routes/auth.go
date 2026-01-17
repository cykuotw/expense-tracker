package route

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func initThirdParty() {
	// OAuth 2.0 setup
	store := sessions.NewCookieStore([]byte(config.Envs.ThirdPartySessionSecret))
	store.MaxAge(int(config.Envs.ThirdPartySessionMaxAge))
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = false // set to true when https

	gothic.Store = store
	goth.UseProviders(
		google.New(config.Envs.GoogleClientId,
			config.Envs.GoogleClientSecret,
			config.Envs.GoogleCallbackUrl,
			"profile", "email"),
	)
}

func (h *Handler) handleThirdParty(c *gin.Context) error {
	provider := c.Param("provider")

	query := c.Request.URL.Query()
	query.Add("provider", provider)
	c.Request.URL.RawQuery = query.Encode()

	if _, err := gothic.CompleteUserAuth(c.Writer, c.Request); err == nil {
		c.Redirect(http.StatusTemporaryRedirect, "/register")
	} else {
		gothic.BeginAuthHandler(c.Writer, c.Request)
	}

	return nil
}

func (h *Handler) handleThirdPartyCallback(c *gin.Context) error {
	provider := c.Param("provider")

	query := c.Request.URL.Query()
	query.Add("provider", provider)
	c.Request.URL.RawQuery = query.Encode()

	gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		fmt.Println(err)
		c.Status(http.StatusInternalServerError)
		return err
	}

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
	token, err := auth.CreateJWT(secret, user.ID)
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

	c.SetCookie(
		"access_token", token,
		int(config.Envs.JWTExpirationInSeconds),
		"/", "localhost", false, true)
	c.SetCookie(
		"refresh_token", refreshToken,
		int(config.Envs.RefreshJWTExpirationInSeconds),
		"/", "localhost", false, true)

	frontendUrl := fmt.Sprintf("http://%s", config.Envs.FrontendReactURL)
	c.Redirect(http.StatusTemporaryRedirect, frontendUrl)

	return nil
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
