package requestserver

import (
	"errors"
	"expense-tracker/backend/config"
	"testing"
)

type fakeDatabase struct{}

func (*fakeDatabase) Ping() error  { return nil }
func (*fakeDatabase) Close() error { return nil }

func TestOpenDatabaseRejectsConfigurationBeforeOpeningDatabase(t *testing.T) {
	opened := false
	_, err := OpenDatabase(config.Config{Mode: "release"}, config.RequestServerStandalone, func(config.Config) (*fakeDatabase, error) {
		opened = true
		return nil, errors.New("must not be called")
	})
	if err == nil {
		t.Fatal("expected invalid release configuration")
	}
	if opened {
		t.Fatal("database constructor was called before configuration validation")
	}
}
