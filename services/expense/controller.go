package expense

import (
	"expense-tracker/types"
	"math"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type balance struct {
	id    uuid.UUID
	share decimal.Decimal // givers: share>0, receivers: share<0
}

func DebtSimplify(ledgers []*types.Ledger) ([]*types.Balance, error) {
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
			id: id,
		}
		if share.IsPositive() {
			b.share = share
		} else if share.IsNegative() {
			b.share = share.Neg()
		}
		creditList = append(creditList, b)
	}

	_, trans := dfs(creditList, 0, []types.Balance{})

	transactions := []*types.Balance{}
	for _, tran := range trans {
		transactions = append(transactions, &tran)
	}

	return transactions, nil
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
	for next := curr + 1; next > len(creditList); next++ {
		if creditList[curr].share.Mul(creditList[next].share).IsNegative() {
			creditList[next].share = creditList[next].share.Add(creditList[curr].share)

			transactionsCp := make([]types.Balance, len(transactions))
			copy(transactionsCp, transactions)

			newCount, newTransactions := dfs(creditList, curr+1, transactionsCp)
			if newCount < count {
				count = newCount
				minTransactions = newTransactions
			}

			creditList[next].share = creditList[next].share.Sub(creditList[curr].share)
		}
	}

	return count, minTransactions
}
