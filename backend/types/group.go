package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type GroupStore interface {
	CreateGroup(group Group) error

	GetGroupByID(id string) (*Group, error)
	GetGroupListByUser(userid string) ([]GetGroupListResponse, error)
	GetGroupMemberByGroupID(groupId string) ([]*User, error)
	GetGroupByIDAndUser(groupID string, userID string) (*Group, error)
	GetGroupCurrency(groupID string) (string, error)
	GetRelatedUser(currentUser string, groupId string) ([]*RelatedMember, error)

	UpdateGroupMember(action string, userid string, groupID string) error
	UpdateGroupStatus(groupID string, creatorID string, isActive bool) error
	UpdateGroup(group Group) error

	CheckGroupExistById(id string) (bool, error)
	CheckGroupUserPairExist(groupId string, userId string) (bool, error)
}

type Group struct {
	ID           uuid.UUID `json:"id"`
	GroupName    string    `json:"groupName"`
	Description  string    `json:"description"`
	CreateTime   time.Time `json:"createTime"`
	IsActive     bool      `json:"isActive"`
	Currency     string    `json:"currency"`
	GroupType    string    `json:"groupType"`
	CreateByUser uuid.UUID `json:"createByUser"`
}

type CreateGroupPayload struct {
	GroupName   string `json:"groupName"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
	GroupType   string `json:"groupType"`
}

type UpdateGroupPayload struct {
	GroupName   string `json:"groupName"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
	GroupType   string `json:"groupType"`
}

type UpdateGroupMemberPayload struct {
	Action  string `json:"action"`
	UserID  string `json:"userId"`
	GroupID string `json:"groupId"`
}

// ReplaceGroupMembersPayload replaces a group's membership in one mutation.
// The authenticated group creator is always retained by the server.
type ReplaceGroupMembersPayload struct {
	GroupID   string   `json:"groupId"`
	MemberIDs []string `json:"memberIds"`
}

type GetGroupResponse struct {
	GroupName   string        `json:"groupName"`
	Description string        `json:"description"`
	Currency    string        `json:"currency"`
	GroupType   string        `json:"groupType"`
	Members     []GroupMember `json:"members"`
}

type GroupMember struct {
	UserID   string `json:"userId"`
	Username string `json:"username"` // username or email
}

type RelatedMember struct {
	UserID       string `json:"userId"`
	Username     string `json:"username"` // username or email
	ExistInGroup bool   `json:"existInGroup"`
}

type GroupBalanceStatus string

const (
	GroupBalanceStatusSettled GroupBalanceStatus = "settled"
	GroupBalanceStatusOwed    GroupBalanceStatus = "owed"
	GroupBalanceStatusOwing   GroupBalanceStatus = "owing"
)

type GetGroupListResponse struct {
	ID            string             `json:"id"`
	GroupName     string             `json:"groupName"`
	Description   string             `json:"description"`
	Currency      string             `json:"currency"`
	GroupType     string             `json:"groupType"`
	BalanceStatus GroupBalanceStatus `json:"balanceStatus"`
	BalanceAmount decimal.Decimal    `json:"balanceAmount"`
}
