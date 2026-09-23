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
	Currency       string
}

type BalanceRsp struct {
	ID               uuid.UUID       `json:"id"`
	SenderUserID     uuid.UUID       `json:"senderUserId"`
	SenderUesrname   string          `json:"senderUsername"`
	ReceiverUserID   uuid.UUID       `json:"receiverUserId"`
	ReceiverUsername string          `json:"receiverUsername"`
	Balance          decimal.Decimal `json:"balance"`
	Currency         string          `json:"currency"`
}

type BalanceResponse struct {
	Currency          string             `json:"currency"`
	CurrentUser       string             `json:"currentUser"`
	Balances          []BalanceRsp       `json:"balances"`
	SettlementPreview *SettlementPreview `json:"settlementPreview"`
}

type SettlementPreviewContribution struct {
	BalanceID      uuid.UUID        `json:"balanceId"`
	SourceCurrency string           `json:"sourceCurrency"`
	SourceAmount   decimal.Decimal  `json:"sourceAmount"`
	Rate           *decimal.Decimal `json:"rate"`
	PreviewAmount  *decimal.Decimal `json:"previewAmount"`
}

type SettlementPreview struct {
	Currency              string                          `json:"currency"`
	Complete              bool                            `json:"complete"`
	NetAmount             *decimal.Decimal                `json:"netAmount"`
	Contributions         []SettlementPreviewContribution `json:"contributions"`
	MissingRateCurrencies []string                        `json:"missingRateCurrencies"`
}
