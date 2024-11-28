package expense

import (
	"expense-tracker/types"
	"math"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Controller struct{}

func NewController() *Controller {
	return &Controller{}
}

type balance struct {
	id    uuid.UUID
	share decimal.Decimal // givers: share>0, receivers: share<0
}

func (c *Controller) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	balanceMap := map[uuid.UUID]decimal.Decimal{}

	for _, ledger := range ledgers {
		val, ok := balanceMap[ledger.LenderUserID]
		if ok {
			balanceMap[ledger.LenderUserID] = val.Sub(ledger.Share)
		} else {
			balanceMap[ledger.LenderUserID] = ledger.Share.Neg()
		}

		val, ok = balanceMap[ledger.BorrowerUesrID]
		if ok {
			balanceMap[ledger.BorrowerUesrID] = val.Add(ledger.Share)
		} else {
			balanceMap[ledger.BorrowerUesrID] = ledger.Share
		}
	}

	creditList := []balance{}

	for id, share := range balanceMap {
		if share.IsZero() {
			continue
		}
		b := balance{
			id:    id,
			share: share,
		}
		creditList = append(creditList, b)
	}

	_, trans := dfs(creditList, 0, []types.Balance{})

	transactions := []*types.Balance{}
	for _, tran := range trans {
		if tran.Share.IsNegative() {
			tmpID := tran.ReceiverUserID
			tran.ReceiverUserID = tran.SenderUserID
			tran.SenderUserID = tmpID
			tran.Share = tran.Share.Neg()
		}
		transactions = append(transactions, &tran)
	}

	return transactions
}

func dfs(creditList []balance, curr int, transactions []types.Balance) (int, []types.Balance) {
	for curr < len(creditList) && creditList[curr].share.IsZero() {
		curr++
	}

	if curr == len(creditList) {
		return 0, transactions
	}

	count := math.MaxInt
	minTransactions := []types.Balance{}
	for next := curr + 1; next < len(creditList); next++ {
		if creditList[curr].share.Mul(creditList[next].share).IsPositive() {
			continue
		}

		transactionsCp := make([]types.Balance, len(transactions))
		copy(transactionsCp, transactions)
		trans := types.Balance{
			SenderUserID:   creditList[curr].id,
			ReceiverUserID: creditList[next].id,
			Share:          creditList[curr].share,
		}
		transactionsCp = append(transactionsCp, trans)

		originReceiverShare := creditList[next].share
		creditList[next].share = originReceiverShare.Add(creditList[curr].share)
		newCount, newTransactions := dfs(creditList, curr+1, transactionsCp)

		if count > newCount+1 {
			count = newCount + 1
			minTransactions = newTransactions
		}

		creditList[next].share = originReceiverShare
	}

	return count, minTransactions
}
