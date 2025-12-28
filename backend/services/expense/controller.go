package expense

import (
	"expense-tracker/backend/types"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Controller struct{}

func NewController() *Controller { return &Controller{} }

type balance struct {
	id    uuid.UUID
	share decimal.Decimal // >0 pays, <0 receives
}

func (c *Controller) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	// 1) Build net per user
	balanceMap := map[uuid.UUID]decimal.Decimal{}

	for _, l := range ledgers {
		// lender is creditor => net decreases
		if v, ok := balanceMap[l.LenderUserID]; ok {
			balanceMap[l.LenderUserID] = v.Sub(l.Share)
		} else {
			balanceMap[l.LenderUserID] = l.Share.Neg()
		}

		// borrower is debtor => net increases
		if v, ok := balanceMap[l.BorrowerUesrID]; ok {
			balanceMap[l.BorrowerUesrID] = v.Add(l.Share)
		} else {
			balanceMap[l.BorrowerUesrID] = l.Share
		}
	}

	// 2) Split into debtors / creditors
	debtors := make([]balance, 0)
	creditors := make([]balance, 0)

	for id, share := range balanceMap {
		if share.IsZero() {
			continue
		}
		if share.IsPositive() {
			debtors = append(debtors, balance{id: id, share: share})
		} else {
			creditors = append(creditors, balance{id: id, share: share}) // negative
		}
	}

	// 3) Deterministic order (tie-breaker)
	sort.Slice(debtors, func(i, j int) bool {
		return debtors[i].id.String() < debtors[j].id.String()
	})
	sort.Slice(creditors, func(i, j int) bool {
		return creditors[i].id.String() < creditors[j].id.String()
	})

	// 4) Greedy match
	now := time.Now()
	out := make([]*types.Balance, 0, len(debtors)+len(creditors))

	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		d := &debtors[i]    // >0
		cr := &creditors[j] // <0

		needPay := d.share
		needRecv := cr.share.Neg() // abs

		amt := decimal.Min(needPay, needRecv)
		if amt.IsZero() {
			// Defensive: avoid infinite loop if weird precision residue appears
			// Move pointers when one side is effectively done.
			if d.share.IsZero() {
				i++
			}
			if cr.share.IsZero() {
				j++
			}
			continue
		}

		tx := &types.Balance{
			ID:             uuid.New(),
			SenderUserID:   d.id,
			ReceiverUserID: cr.id,
			Share:          amt,
			CreateTime:     now,
			UpdateTime:     now,
			IsOutdated:     false,
			IsSettled:      false,
			// GroupID / SettledTime 留空由上層填
		}
		out = append(out, tx)

		d.share = d.share.Sub(amt)   // toward 0
		cr.share = cr.share.Add(amt) // negative toward 0

		if d.share.IsZero() {
			i++
		}
		if cr.share.IsZero() {
			j++
		}
	}

	// 5) Stable output order
	sort.Slice(out, func(i, j int) bool {
		si, sj := out[i].SenderUserID.String(), out[j].SenderUserID.String()
		if si != sj {
			return si < sj
		}
		ri, rj := out[i].ReceiverUserID.String(), out[j].ReceiverUserID.String()
		if ri != rj {
			return ri < rj
		}
		return out[i].Share.Cmp(out[j].Share) < 0
	})

	return out
}
