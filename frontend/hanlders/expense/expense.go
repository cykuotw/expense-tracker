package expense

import (
	"encoding/json"
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/components"
	"expense-tracker/frontend/views/index"
	"expense-tracker/types"
	"fmt"
	"math"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/create_expense", common.Make(h.handleCreateNewExpenseGet))
	router.POST("/create_expense", common.Make(h.handleCreateNewExpensePost))
	router.GET("/expense/:expenseId", common.Make(h.handleGetExpenseDetail))
	router.GET("/expense/:expenseId/edit", common.Make(h.handleGetExpenseEdit))
	router.PUT("/expense/:expenseId/delete", common.Make(h.handleGetExpenseDelete))
	router.PUT("/update_expense", common.Make(h.handleUpdateExpense))
	router.GET("/expense_types", common.Make(h.handleGetExpenseType))
	router.GET("/expense_types/:select", common.Make(h.handleGetExpenseType))
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

	var form common.ExpenseForm
	if err := c.ShouldBind(&form); err != nil {
		c.Status(http.StatusBadRequest)
	}

	// TODO: use DB to do this
	precisionMap := map[string]int{
		"CAD": 2,
		"USD": 2,
		"NTD": 0,
	}

	switch form.SpliteRule {
	case common.Equally.String(), common.YouHalf.String(), common.OtherHalf.String():
		peopleCount := len(form.Borrowers)
		precision := precisionMap[form.Currency]

		split := float32(math.Floor(float64(form.Total)/float64(peopleCount)*(math.Pow10(precision))) / math.Pow10(precision))
		remaining := form.Total - (split * (float32(peopleCount - 1)))

		randIndex := rand.Intn(len(form.Shares))
		for i := range peopleCount {
			if i == randIndex {
				form.Shares[i] = remaining
			} else {
				form.Shares[i] = split
			}
		}

	case common.YouFull.String():
		form.Shares[1] = form.Total

	case common.OtherFull.String():
		form.Shares[0] = form.Total
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
		SplitRule:     form.SpliteRule,
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

func (h *Handler) handleGetExpenseDetail(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	expenseID := c.Param("expenseId")

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/expense/"+expenseID, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	resPayload := types.ExpenseResponse{}
	err = json.NewDecoder(res.Body).Decode(&resPayload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	return common.Render(c.Writer, c.Request, index.ExpenseDetail(resPayload))
}

func (h *Handler) handleGetExpenseEdit(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	expenseID := c.Param("expenseId")

	res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/expense/"+expenseID, token, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer res.Body.Close()

	resPayload := types.ExpenseResponse{}
	err = json.NewDecoder(res.Body).Decode(&resPayload)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	return common.Render(c.Writer, c.Request, index.EditExpense(resPayload))
}

func (h *Handler) handleGetExpenseDelete(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	expenseID := c.Param("expenseId")
	groupId := c.PostForm("groupId")

	// Make a request to delete the expense
	res, err := common.MakeBackendHTTPRequest(http.MethodPut, "/delete_expense/"+expenseID, token, nil)
	if err != nil {
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
		c.Status(http.StatusInternalServerError)
		return fmt.Errorf(resErr.Error)
	}

	c.Header("HX-Redirect", "/group/"+groupId)
	c.Status(http.StatusOK)

	return nil
}

func (h *Handler) handleUpdateExpense(c *gin.Context) error {
	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	expenseID := c.Query("e")

	var form common.ExpenseForm
	if err := c.ShouldBind(&form); err != nil {
		c.Status(http.StatusBadRequest)
	}

	precisionMap := map[string]int{
		"CAD": 2,
		"USD": 2,
		"NTD": 0,
	}

	switch form.SpliteRule {
	case common.Equally.String(), common.YouHalf.String(), common.OtherHalf.String():
		peopleCount := len(form.Borrowers)
		precision := precisionMap[form.Currency]

		split := float32(math.Floor(float64(form.Total)/float64(peopleCount)*(math.Pow10(precision))) / math.Pow10(precision))
		remaining := form.Total - (split * (float32(peopleCount - 1)))

		randIndex := rand.Intn(len(form.Shares))
		for i := range peopleCount {
			if i == randIndex {
				form.Shares[i] = remaining
			} else {
				form.Shares[i] = split
			}
		}

	case common.YouFull.String():
		form.Shares[0] = 0
		form.Shares[1] = form.Total

	case common.OtherFull.String():
		form.Shares[0] = form.Total
		form.Shares[1] = 0
	}

	len := len(form.Borrowers)
	ledgerList := []types.LedgerUpdatePayload{}
	for i := 0; i < len; i++ {
		ledger := types.LedgerUpdatePayload{
			ID: form.Ids[i],
			LedgerPayload: types.LedgerPayload{
				BorrowerUesrID: form.Borrowers[i],
				LenderUserID:   form.Payer,
				Share:          decimal.NewFromFloat32(form.Shares[i]),
			},
		}
		ledgerList = append(ledgerList, ledger)
	}

	payload := types.ExpenseUpdatePayload{
		Description:   form.Description,
		GroupID:       uuid.MustParse(form.GroupId),
		PayByUserId:   form.Payer,
		ExpenseTypeID: uuid.MustParse(form.ExpenseTypeID),
		Total:         decimal.NewFromFloat32(form.Total),
		Currency:      form.Currency,
		Ledgers:       ledgerList,
		SplitRule:     form.SpliteRule,

		// ProviderName: , 	// TODO: image reconition feature
		// SubTotal: , 		// TODO: image reconition feature
		// TaxFeeTip: , 	// TODO: image reconition feature
		// InvoicePicUrl: , // TODO: image reconition feature
		// Items: , 		// TODO: image reconition feature
	}

	res, err := common.MakeBackendHTTPRequest(http.MethodPut, "/expense/"+expenseID, token, payload)
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

	c.Header("HX-Redirect", "/expense/"+expenseID)
	c.Status(200)

	return nil
}

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

func (h *Handler) handleGetSplitRules(c *gin.Context) error {
	handleGroup := func(token string, groupId string) error {
		res, err := common.MakeBackendHTTPRequest(http.MethodGet, "/group_member/"+groupId, token, nil)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		defer res.Body.Close()

		members := []types.GroupMember{}
		err = json.NewDecoder(res.Body).Decode(&members)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}

		currUser := members[len(members)-1]

		return common.Render(c.Writer, c.Request, components.SplitRule(currUser, members))
	}

	handleExpense := func(token string, expenseId string) error {
		// get expense
		resLedger, err := common.MakeBackendHTTPRequest(http.MethodGet, "/expense/"+expenseId, token, nil)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		defer resLedger.Body.Close()

		expense := types.ExpenseResponse{}
		err = json.NewDecoder(resLedger.Body).Decode(&expense)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}

		mpLedger := map[string]types.LedgerResponse{}
		for _, ledger := range expense.Ledgers {
			mpLedger[ledger.BorrowerUserId] = ledger
		}

		lenderId := expense.Ledgers[0].LenderUserId

		// evaluate split rule
		var splitRule common.SplitOption
		switch expense.SplitRule {
		case "Unequally":
			splitRule = common.Unequally
		case "You-Half":
			splitRule = common.YouHalf
		case "You-Full":
			splitRule = common.YouFull
		case "Other-Half":
			splitRule = common.OtherHalf
		case "Other-Full":
			splitRule = common.OtherFull
		default:
			splitRule = common.Equally
		}

		// get group members
		resGroupMember, err := common.MakeBackendHTTPRequest(http.MethodGet, "/group_member/"+expense.GroupId, token, nil)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		defer resGroupMember.Body.Close()

		members := []types.GroupMember{}
		err = json.NewDecoder(resGroupMember.Body).Decode(&members)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}

		currUser := members[len(members)-1]

		return common.Render(c.Writer, c.Request,
			components.SplitRuleWithLedger(currUser, members, mpLedger, splitRule, lenderId))
	}

	token, err := c.Cookie("access_token")
	if err != nil {
		c.Status(http.StatusUnauthorized)
		c.Writer.Write([]byte("Unauthorized"))
		return err
	}

	groupId := c.Query("g")
	if groupId != "" {
		return handleGroup(token, groupId)
	}

	expenseId := c.Query("e")
	if expenseId != "" {
		return handleExpense(token, expenseId)
	}

	return nil
}
