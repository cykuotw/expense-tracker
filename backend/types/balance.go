package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Balance struct {
	ID             uuid.UUID
	SenderUserID   uuid.UUID
	ReceiverUserID uuid.UUID
	Share          decimal.Decimal
	GroupID        uuid.UUID
	CreateTime     time.Time
	IsOutdated     bool
	UpdateTime     time.Time
	IsSettled      bool
	SettledTime    time.Time
}

type BalanceRsp struct {
	ID               uuid.UUID       `json:"id"`
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
