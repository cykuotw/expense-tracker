package main

import (
	"io"
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedMigrationSource(t *testing.T) {
	source, err := iofs.New(migrations, "migrations")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, source.Close()) })
	version, err := source.First()
	require.NoError(t, err)
	require.Equal(t, uint(1), version)
	// Both directions must be bundled in the executable, regardless of cwd.
	for _, read := range []func(uint) (io.ReadCloser, string, error){source.ReadUp, source.ReadDown} {
		reader, name, err := read(version)
		require.NoError(t, err)
		require.Equal(t, "add_agent_state", name)
		contents, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NotEmpty(t, contents)
		require.NoError(t, reader.Close())
	}
}
