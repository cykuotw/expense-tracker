package types

import (
	"time"

	"github.com/google/uuid"
)

type GroupStore interface {
	CreateGroup(group Group) error

	GetGroupByID(id string) (*Group, error)
	GetGroupListByUser(userid string) ([]*Group, error)
	GetGroupMemberByGroupID(groupId string) ([]*User, error)
	GetGroupByIDAndUser(groupID string, userID string) (*Group, error)
	GetGroupCurrency(groupID string) (string, error)

	UpdateGroupMember(action string, userid string, groupID string) error
	UpdateGroupStatus(groupid string, isActive bool) error
}

type Group struct {
	ID           uuid.UUID `json:"id"`
	GroupName    string    `json:"groupName"`
	Description  string    `json:"description"`
	CreateTime   time.Time `json:"createTime"`
	IsActive     bool      `json:"isActive"`
	Currency     string    `json:"currency"`
	CreateByUser uuid.UUID `json:"createByUser"`
}

type CreateGroupPayload struct {
	GroupName   string `json:"groupName"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
}

type UpdateGroupMemberPayload struct {
	Action  string `json:"action"`
	UserID  string `json:"userId"`
	GroupID string `json:"groupId"`
}

type GetGroupResponse struct {
	GroupName   string        `json:"groupName"`
	Description string        `json:"description"`
	Currency    string        `json:"currency"`
	Members     []GroupMember `json:"members"`
}

type GroupMember struct {
	UserID   string `json:"userId"`
	Username string `json:"username"` // username or email
}

type GetGroupListResponse struct {
	ID          string `json:"id"`
	GroupName   string `json:"groupName"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
}
