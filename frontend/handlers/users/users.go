package users

import (
	"encoding/json"
	"expense-tracker/frontend/handlers/common"
	"expense-tracker/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/user_info", common.Make(h.handleGetUserInfo))
}

func (h *Handler) handleGetUserInfo(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/user_info", token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	payload := types.UserInfoResponse{}
	err = json.NewDecoder(res.Body).Decode(&payload)

	html := `
		<details open class="dropdown">
			<summary>
				<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" fill="currentColor" class="bi bi-person-fill" viewBox="0 0 16 16">
					<path d="M3 14s-1 0-1-1 1-4 6-4 6 3 6 4-1 1-1 1zm5-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6"></path>
				</svg>
				<div class="flex lg:hidden w-2"></div>
				<div class="hidden lg:flex text-lg">
					Account
				</div>
			</summary>
			<ul class="p-2 bg-base-100 rounded-t-none">
				<li class="menu-title text-md text-black">Hi ` + payload.Nickname + `</li>
				<li><a>Link 1</a></li>
				<li><a>Link 2</a></li>
			</ul>
		</details>
	`

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}
