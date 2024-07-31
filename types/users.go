package types

import (
	"time"

	"github.com/google/uuid"
)

type UserStore interface {
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	GetUsernameByID(userid string) (string, error)

	CreateUser(user User) error
	CheckEmailExist(email string) (bool, error)
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
}

type RegisterUserPayload struct {
	Nickname  string `json:"nickname"`
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
}

type LoginUserPayload struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
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
