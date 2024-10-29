package auth

import (
	"expense-tracker/config"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/auth"
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
			config.Envs.GoogleCallbackUrl, "profile"),
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

	fmt.Printf("%+v\n", gothUser)

	c.Redirect(http.StatusTemporaryRedirect, "/")

	return nil
}
