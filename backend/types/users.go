package types

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserStore interface {
	GetUserByEmail(email string) (*User, error)
	GetUserByExternalIdentity(externalType string, externalID string) (*User, error)
	GetUserByID(id string) (*User, error)
	GetUsernameByID(userid string) (string, error)

	CreateUser(user User) error

	CheckEmailExist(email string) (bool, error)
	CheckUserExistByEmail(email string) (bool, error)
	CheckUserExistByID(id string) (bool, error)
	CheckUserExistByUsername(username string) (bool, error)
}

type RegistrationStore interface {
	CreateInvitedUser(ctx context.Context, token string, user User) error
}

type User struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Firstname      string    `json:"firstname"`
	Lastname       string    `json:"lastname"`
	Email          string    `json:"email"`
	Nickname       string    `json:"nickname"`
	PasswordHashed string    `json:"passwordHashed"`
	ExternalType   string    `json:"externalType"`
	ExternalID     string    `json:"externalId"`
	CreateTime     time.Time `json:"createTime"`
	IsActive       bool      `json:"isActive"`
	Role           string    `json:"role"`
}

type RegisterUserPayload struct {
	Nickname  string `json:"nickname"`
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	Token     string `json:"token" validate:"required"`
}

type RegisterGooglePayload struct {
	Token string `json:"token" validate:"required"`
}

type LoginUserPayload struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type ThirdPartyUserPayload struct {
	Nickname     string `json:"nickname"`
	Firstname    string `json:"firstname"`
	Lastname     string `json:"lastname"`
	Email        string `json:"email"`
	ExternalId   string `json:"externalId"`
	ExternalType string `json:"externalType"`
}

type VerifiedGoogleClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Name          string `json:"name,omitempty"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type UserInfoResponse struct {
	Nickname  string `json:"nickname"`
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
}

type AccountResponse struct {
	Nickname              string `json:"nickname"`
	Firstname             string `json:"firstname"`
	Lastname              string `json:"lastname"`
	Email                 string `json:"email"`
	GoogleConnected       bool   `json:"googleConnected"`
	PasswordChangeAllowed bool   `json:"passwordChangeAllowed"`
}

type UpdateOwnProfilePayload struct {
	Nickname  string `json:"nickname" validate:"max=100"`
	Firstname string `json:"firstname" validate:"required,max=100"`
	Lastname  string `json:"lastname" validate:"required,max=100"`
}

type ChangeOwnPasswordPayload struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required"`
}

type AdminUserResponse struct {
	ID         uuid.UUID `json:"id"`
	Firstname  string    `json:"firstname"`
	Lastname   string    `json:"lastname"`
	Email      string    `json:"email"`
	Nickname   string    `json:"nickname"`
	Role       string    `json:"role"`
	IsActive   bool      `json:"isActive"`
	CreateTime time.Time `json:"createTime"`
}

type UpdateUserStatusPayload struct {
	IsActive *bool `json:"isActive" validate:"required"`
}

type UpdateUserRolePayload struct {
	Role string `json:"role" validate:"required,oneof=admin user"`
}
