package user

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FirstAdminStore interface {
	ReconcileFirstAdmin(candidate *types.User, normalizedEmail string) (BootstrapStatus, error)
}

type BootstrapStatus string

const (
	BootstrapStatusCreated       BootstrapStatus = "created"
	BootstrapStatusReconciled    BootstrapStatus = "reconciled"
	BootstrapStatusAlreadyExists BootstrapStatus = "already_exists"
	BootstrapStatusNotRequested  BootstrapStatus = "not_requested"
)

type FirstAdminInput struct {
	Email     string
	Password  string
	Firstname string
	Lastname  string
	Nickname  string
}

type BootstrapDeps struct {
	Now     func() time.Time
	NewUUID func() uuid.UUID
}

func BootstrapFirstAdmin(store FirstAdminStore, input *FirstAdminInput, deps BootstrapDeps) (BootstrapStatus, error) {
	if input == nil {
		return store.ReconcileFirstAdmin(nil, "")
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.NewUUID == nil {
		deps.NewUUID = uuid.New
	}

	normalized, err := normalizeFirstAdminInput(*input)
	if err != nil {
		return "", err
	}

	hashedPassword, err := auth.HashPassword(normalized.Password)
	if err != nil {
		return "", err
	}

	username := normalized.Nickname
	if username == "" {
		username = normalized.Firstname + " " + normalized.Lastname
	}

	candidate := &types.User{
		ID:             deps.NewUUID(),
		Username:       username,
		Nickname:       normalized.Nickname,
		Firstname:      normalized.Firstname,
		Lastname:       normalized.Lastname,
		Email:          normalized.Email,
		PasswordHashed: hashedPassword,
		ExternalType:   "",
		ExternalID:     "",
		CreateTime:     deps.Now(),
		IsActive:       true,
		Role:           "admin",
	}
	return store.ReconcileFirstAdmin(candidate, normalized.Email)
}

func normalizeFirstAdminInput(input FirstAdminInput) (FirstAdminInput, error) {
	input.Email = strings.TrimSpace(input.Email)
	input.Password = strings.TrimSpace(input.Password)
	input.Firstname = strings.TrimSpace(input.Firstname)
	input.Lastname = strings.TrimSpace(input.Lastname)
	input.Nickname = strings.TrimSpace(input.Nickname)

	switch {
	case input.Email == "":
		return FirstAdminInput{}, fmt.Errorf("FIRST_ADMIN_EMAIL is required")
	case input.Password == "":
		return FirstAdminInput{}, fmt.Errorf("FIRST_ADMIN_PASSWORD is required")
	case len(input.Password) < 8:
		return FirstAdminInput{}, fmt.Errorf("FIRST_ADMIN_PASSWORD must be at least 8 characters")
	case input.Firstname == "":
		return FirstAdminInput{}, fmt.Errorf("FIRST_ADMIN_FIRSTNAME is required")
	case input.Lastname == "":
		return FirstAdminInput{}, fmt.Errorf("FIRST_ADMIN_LASTNAME is required")
	}

	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email {
		return FirstAdminInput{}, fmt.Errorf("FIRST_ADMIN_EMAIL must be a valid email address")
	}
	input.Email = auth.NormalizeEmail(address.Address)

	return input, nil
}
