package expense

import (
	"encoding/json"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/components"
	"expense-tracker/frontend/views/index"
	"expense-tracker/types"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/create_expense", common.Make(h.handleCreateNewExpenseGet))
	router.POST("/create_expense", common.Make(h.handleCreateNewExpensePost))
	router.GET("/expense_types", common.Make(h.handleGetExpenseType))
	router.GET("/split_rules", common.Make(h.handleGetSplitRules))
}

func (h *Handler) handleCreateNewExpenseGet(c *gin.Context) error {
	groupId := c.Query("g")
	isSubmit := c.Query("submit") == "true"

	return common.Render(c.Writer, c.Request, index.NewExpense(groupId, isSubmit))
}

func (h *Handler) handleCreateNewExpensePost(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	groupId := c.Query("g")

	var form struct {
		Description   string    `form:"description" binding:"required"`
		Payer         string    `form:"payer" binding:"required"`
		ExpenseTypeID string    `form:"expenseType" binding:"required"`
		Total         float32   `form:"total" binding:"required"`
		Currency      string    `form:"currency" binding:"required"`
		Borrowers     []string  `form:"ledger.borrower[]" binding:"required"`
		Shares        []float32 `form:"ledger.share[]" binding:"required"`
	}

	if err := c.ShouldBind(&form); err != nil {
		c.Status(http.StatusBadRequest)
	}

	len := len(form.Borrowers)
	ledgerList := []types.LedgerPayload{}
	for i := 0; i < len; i++ {
		if form.Shares[i] == 0 {
			continue
		}

		ledger := types.LedgerPayload{
			BorrowerUesrID: form.Borrowers[i],
			LenderUserID:   form.Payer,
			Share:          decimal.NewFromFloat32(form.Shares[i]),
		}
		ledgerList = append(ledgerList, ledger)
	}

	payload := types.ExpensePayload{
		Description:   form.Description,
		GroupID:       groupId,
		PayByUserId:   form.Payer,
		ExpenseTypeID: form.ExpenseTypeID,
		Total:         decimal.NewFromFloat32(form.Total),
		Currency:      form.Currency,
		Ledgers:       ledgerList,
		// CreateByUserID: , // filled in backend

		// ProviderName: , 	// TODO: image reconition feature
		// SubTotal: , 		// TODO: image reconition feature
		// TaxFeeTip: , 	// TODO: image reconition feature
		// InvoicePicUrl: , // TODO: image reconition feature
		// Items: , 		// TODO: image reconition feature
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodPost, "/create_expense", token, payload)
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

	c.Header("HX-Redirect", "/create_expense?g="+groupId+"&submit=true")
	c.Status(200)

	return nil
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
