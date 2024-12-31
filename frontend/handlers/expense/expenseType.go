package expense

import (
	"encoding/json"
	"expense-tracker/frontend/handlers/common"
	"expense-tracker/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetExpenseType(c *gin.Context) error {
	selectedType := c.Param("select")
	if selectedType == "" {
		selectedType = "General"
	}

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
		if payload.Name == selectedType {
			tmp := "<option disabled> ----- " + payload.Category + " ----- </option>"
			tmp += "<option selected value=\"" + payload.ID + "\">" + payload.Name + "</option>"
			html = tmp + html
		} else {
			_, ok := set[payload.Category]
			if !ok {
				set[payload.Category] = true
				html += "<option disabled> ----- " + payload.Category + " ----- </option>"
			}
			html += "<option value=\"" + payload.ID + "\">" + payload.Name + "</option>"
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
