package group

import (
	"bytes"
	"encoding/json"
	"expense-tracker/config"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/index"
	"expense-tracker/types"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/create_group", common.Make(h.handleCreateNewGroupGet))
	router.POST("/create_group", common.Make(h.handleCreateNewGroup))
}

func (h *Handler) handleCreateNewGroupGet(c *gin.Context) error {
	return common.Render(c.Writer, c.Request, index.NewGroup())
}

func (h *Handler) handleCreateNewGroup(c *gin.Context) error {
	apiUrl := "http://" + config.Envs.BackendURL + config.Envs.APIPath

	payload := types.CreateGroupPayload{
		GroupName:   c.PostForm("groupname"),
		Description: c.PostForm("description"),
		Currency:    c.PostForm("currency"),
	}

	marshalled, err := json.Marshal(payload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	req, err := http.NewRequest(http.MethodPost, apiUrl+"/create_group", bytes.NewBuffer(marshalled))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		resErr := types.ServerErr{}
		err = json.NewDecoder(res.Body).Decode(&resErr)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		c.Status(http.StatusInternalServerError)
		return fmt.Errorf(resErr.Error)
	}

	c.Header("HX-Redirect", "/")
	c.Status(200)

	return nil
}
