package types

import (
	"time"

	"github.com/google/uuid"
)

type UserStore interface {
	GetUserByEmail(email string) (*User, error)
	GetUserByUsername(username string) (*User, error)

	GetUserByID(id string) (*User, error)
	CreateUser(user User) error
}

type User struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Firstname      string    `json:"firstname"`
	Lastname       string    `json:"lastname"`
	Email          string    `json:"email"`
	PasswordHashed string    `json:"passwordHashed"`
	ExternalType   string    `json:"externalType"`
	ExternalID     string    `json:"externalId"`
	CreateTime     time.Time `json:"createTime"`
	IsActive       bool      `json:"isActive"`
}

type RegisterUserPayload struct {
	Username  string `json:"username" validate:"required"`
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=130"`
}

type LoginUserPayload struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
