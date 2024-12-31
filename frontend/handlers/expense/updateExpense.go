package expense

import (
	"encoding/json"
	"expense-tracker/frontend/handlers/common"
	"expense-tracker/types"
	"fmt"
	"math"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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
		return fmt.Errorf("%s", resErr.Error)
	}

	c.Header("HX-Redirect", "/expense/"+expenseID)
	c.Status(200)

	return nil
}
