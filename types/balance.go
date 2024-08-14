package types

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Balance struct {
	SenderUserID   uuid.UUID
	ReceiverUserID uuid.UUID
	Share          decimal.Decimal
}

type BalanceRsp struct {
	SenderUserID     uuid.UUID       `json:"senderUserId"`
	SenderUesrname   string          `json:"senderUsername"`
	ReceiverUserID   uuid.UUID       `json:"receiverUserId"`
	ReceiverUsername string          `json:"receiverUsername"`
	Balance          decimal.Decimal `json:"balance"`
}

type BalanceResponse struct {
	Currency    string       `json:"currency"`
	CurrentUser string       `json:"currentUser"`
	Balances    []BalanceRsp `json:"balances"`
}
