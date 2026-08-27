package user

import (
	"errors"
	"expense-tracker/backend/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStatusChange(t *testing.T) {
	tests := []struct {
		name         string
		actor        string
		target       string
		role         string
		current      bool
		next         bool
		activeAdmins int
		protected    bool
		expected     error
	}{
		{name: "activate user", actor: "admin", target: "user", role: "user", next: true},
		{name: "disable user", actor: "admin", target: "user", role: "user", current: true, activeAdmins: 1},
		{name: "reject self", actor: "admin", target: "admin", role: "admin", current: true, activeAdmins: 2, expected: types.ErrCannotDeactivateSelf},
		{name: "reject last admin", actor: "other-admin", target: "admin", role: "admin", current: true, activeAdmins: 1, expected: types.ErrLastActiveAdmin},
		{name: "reject protected no-op", actor: "other-admin", target: "owner", role: "admin", current: true, next: true, activeAdmins: 2, protected: true, expected: types.ErrProtectedAdmin},
		{name: "protected takes precedence over self", actor: "owner", target: "owner", role: "admin", current: true, activeAdmins: 1, protected: true, expected: types.ErrProtectedAdmin},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateStatusChange(test.actor, test.target, test.role, test.current, test.next, test.activeAdmins, test.protected)
			assert.True(t, errors.Is(err, test.expected))
		})
	}
}

func TestValidateRoleChange(t *testing.T) {
	tests := []struct {
		name         string
		actor        string
		target       string
		currentRole  string
		nextRole     string
		active       bool
		activeAdmins int
		protected    bool
		expected     error
	}{
		{name: "promote user", actor: "admin", target: "user", currentRole: "user", nextRole: "admin", active: true},
		{name: "demote admin", actor: "admin-1", target: "admin-2", currentRole: "admin", nextRole: "user", active: true, activeAdmins: 2},
		{name: "reject self", actor: "admin", target: "admin", currentRole: "admin", nextRole: "user", active: true, activeAdmins: 2, expected: types.ErrCannotChangeOwnRole},
		{name: "reject last admin", actor: "other", target: "admin", currentRole: "admin", nextRole: "user", active: true, activeAdmins: 1, expected: types.ErrLastActiveAdmin},
		{name: "reject unknown role", actor: "admin", target: "user", currentRole: "user", nextRole: "owner", active: true, expected: types.ErrInvalidUserRole},
		{name: "reject protected no-op", actor: "other-admin", target: "owner", currentRole: "admin", nextRole: "admin", active: true, activeAdmins: 2, protected: true, expected: types.ErrProtectedAdmin},
		{name: "protected takes precedence over self", actor: "owner", target: "owner", currentRole: "admin", nextRole: "user", active: true, activeAdmins: 1, protected: true, expected: types.ErrProtectedAdmin},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRoleChange(test.actor, test.target, test.currentRole, test.nextRole, test.active, test.activeAdmins, test.protected)
			assert.True(t, errors.Is(err, test.expected))
		})
	}
}
