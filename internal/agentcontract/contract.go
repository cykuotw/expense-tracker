// Package agentcontract defines the bounded, read-only Agent/Worker wire contract.
// It has no model, database, or credential-storage dependencies.
package agentcontract

import "time"

const (
	Prefix            = "/internal/agent/v1"
	PreflightPath     = Prefix + "/preflight"
	BalancePath       = Prefix + "/balance/{groupId}"
	AccessTokenHeader = "X-Agent-Access-Token"
	MaxBalanceRows    = 100
	MaxResponseBytes  = 64 << 10
)

type Preflight struct {
	ActorID   string    `json:"actorId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Balance struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type BalanceResult struct {
	GroupID  string    `json:"groupId"`
	Balances []Balance `json:"balances"`
}
