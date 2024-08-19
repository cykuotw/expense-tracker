package expense

import (
	"encoding/json"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/components"
	"expense-tracker/frontend/views/index"
	"expense-tracker/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/create_expense", common.Make(h.handleCreateNewExpenseGet))
	router.GET("/expense_types", common.Make(h.handleGetExpenseType))
	router.GET("/split_rules", common.Make(h.handleGetSplitRules))
}

func (h *Handler) handleCreateNewExpenseGet(c *gin.Context) error {
	groupId := c.Query("g")

	return common.Render(c.Writer, c.Request, index.NewExpense(groupId))
}

func (h *Handler) handleGetExpenseType(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/expense_types", token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	payloadList := []types.ExpenseTypeResponse{}
	err = json.NewDecoder(res.Body).Decode(&payloadList)

	set := map[string]bool{}
	html := ""
	for _, payload := range payloadList {
		if payload.Name == "General" {
			tmp := "<option disabled> ----- " + payload.Category + " ----- </option>"
			tmp += "<option selected>" + payload.Name + "</option>"
			html = tmp + html
		} else {
			_, ok := set[payload.Category]
			if !ok {
				set[payload.Category] = true
				html += "<option disabled> ----- " + payload.Category + " ----- </option>"
			}
			html += "<option>" + payload.Name + "</option>"
		}
	}
	html = `<select
				class="select select-bordered w-full text-base text-center"
				id="expenseType"
				name="expenseType"
			>` + html + "</select>"

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(html))

	return nil
}

func (h *Handler) handleGetSplitRules(c *gin.Context) error {
	groupId := c.Query("g")

	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/group/"+groupId, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	payload := types.GetGroupResponse{}
	err = json.NewDecoder(res.Body).Decode(&payload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	members := payload.Members
	currUser := payload.Members[len(payload.Members)-1]

	return common.Render(c.Writer, c.Request, components.SplitRule(currUser, members))
}
