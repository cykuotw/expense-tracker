package types

import (
	"time"

	"github.com/google/uuid"
)

const WebPushSubscriptionLimit = 5

type WebPushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

type WebPushSubscription struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Endpoint    string
	P256DH      string
	Auth        string
	ShowDetails bool
}

type WebPushSettings struct {
	Enabled      bool                    `json:"enabled"`
	ShowDetails  bool                    `json:"showDetails"`
	VAPIDKey     string                  `json:"vapidPublicKey"`
	MutedGroups  []GroupNotificationMute `json:"mutedGroups"`
	Subscription int                     `json:"subscriptionCount"`
}

type WebPushSubscriptionStatus struct {
	Registered  bool `json:"registered"`
	ShowDetails bool `json:"showDetails"`
}

type GroupNotificationMute struct {
	GroupID   uuid.UUID `json:"groupId"`
	GroupName string    `json:"groupName"`
	Muted     bool      `json:"muted"`
}

type WebPushDelivery struct {
	ID           uuid.UUID
	Subscription WebPushSubscription
	ExpenseID    uuid.UUID
	RecipientID  uuid.UUID
	GroupID      uuid.UUID
	ActorID      uuid.UUID
	GroupName    string
	Currency     string
	Amount       string
	Attempts     int
	ExpiresAt    time.Time
}
