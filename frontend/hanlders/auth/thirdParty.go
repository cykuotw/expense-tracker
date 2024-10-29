package auth

import (
	"encoding/json"
	"expense-tracker/config"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/auth"
	"expense-tracker/types"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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
		google.New(config.Envs.GoogleClientId, config.Envs.GoogleClientSecret,
			config.Envs.GoogleCallbackUrl, "profile", "email"),
	)
}

func (h *Handler) handleThirdParty(c *gin.Context) error {
	provider := c.Param("provider")

	query := c.Request.URL.Query()
	query.Add("provider", provider)
	c.Request.URL.RawQuery = query.Encode()

	if _, err := gothic.CompleteUserAuth(c.Writer, c.Request); err == nil {
		return common.Render(c.Writer, c.Request, auth.Register())
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
		c.Status(http.StatusInternalServerError)
		return err
	}

	// fmt.Printf("%+v\n", gothUser)

	payload := types.ThirdPartyUserPayload{
		Nickname:     gothUser.NickName,
		Firstname:    gothUser.FirstName,
		Lastname:     gothUser.LastName,
		Email:        gothUser.Email,
		ExternalId:   gothUser.UserID,
		ExternalType: gothUser.Provider,
	}
	res, err := common.MakeBackendHTTPRequest(http.MethodPost, "/auth", "", payload)
	if err != nil {
		fmt.Println(err)
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		resErr := types.ServerErr{}
		err = json.NewDecoder(res.Body).Decode(&resErr)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		c.Writer.Write([]byte(resErr.Error))
		c.Status(http.StatusInternalServerError)
		return fmt.Errorf(resErr.Error)
	}

	token := types.LoginResponse{}
	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	c.SetCookie(
		"access_token", token.Token,
		int(config.Envs.JWTExpirationInSeconds),
		"/", "localhost", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")

	return nil
}
