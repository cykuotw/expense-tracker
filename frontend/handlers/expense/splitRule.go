package expense

import (
	"encoding/json"
	"expense-tracker/frontend/handlers/common"
	"expense-tracker/frontend/views/components"
	"expense-tracker/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
