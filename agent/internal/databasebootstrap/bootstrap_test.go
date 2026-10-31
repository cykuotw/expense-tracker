package databasebootstrap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBootstrapRequiresRuntimeCredentialsBeforeMigrations(t *testing.T) {
	for _, config := range []struct{ user, password string }{
		{"", "password"}, {" ", "password"}, {"agent_state_runtime", ""},
	} {
		// No database is needed: invalid setup must fail before touching migrations.
		require.ErrorContains(t, Run(t.Context(), nil, nil, config.user, config.password), "role configuration is required")
	}
}
