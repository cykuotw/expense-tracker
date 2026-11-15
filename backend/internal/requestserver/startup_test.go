package requestserver

import (
	"errors"
	"expense-tracker/backend/config"
	"strings"
	"testing"
)

type fakeDatabase struct{ pings int }

func (db *fakeDatabase) Ping() error { db.pings++; return nil }
func (*fakeDatabase) Close() error   { return nil }

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

func TestServerlessDatabaseOpensWithoutPing(t *testing.T) {
	for key, value := range map[string]string{
		"MODE": "release", "DB_PUBLIC_HOST": "10.0.0.2", "DB_PORT": "5432",
		"DB_USER": "expense_runtime", "DB_PASSWORD": "database-password-not-a-placeholder", "DB_NAME": "expense_tracker", "DB_SSLMODE": "disable",
		"DB_MAX_OPEN_CONNS": "2", "DB_MAX_IDLE_CONNS": "1", "DB_CONN_MAX_LIFETIME_SECONDS": "300", "DB_CONN_MAX_IDLE_TIME_SECONDS": "60",
		"JWT_SECRET": strings.Repeat("a", 32), "JWT_EXP": "900", "REFRESH_JWT_SECRET": strings.Repeat("b", 32), "REFRESH_JWT_EXP": "604800",
		"OCR_CAPABILITY_SECRET": strings.Repeat("c", 32), "EXPENSES_PER_PAGE": "25",
		"FRONTEND_ORIGIN": "https://app.example.com", "CORS_ALLOWED_ORIGINS": "https://app.example.com", "CORS_ALLOW_CREDENTIALS": "true",
		"AUTH_COOKIE_SECURE": "true", "AUTH_COOKIE_SAME_SITE": "lax", "GOOGLE_OAUTH_ENABLED": "false", "GOOGLE_EXCHANGE_MODE": "upstream_verified",
	} {
		t.Setenv(key, value)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	previous := config.Envs
	t.Cleanup(func() { config.Apply(previous) })
	db := &fakeDatabase{}
	storage, err := OpenDatabase(cfg, config.RequestServerServerless, func(config.Config) (*fakeDatabase, error) { return db, nil })
	if err != nil || storage != db || db.pings != 0 {
		t.Fatalf("startup attempted network ping: %v %d", err, db.pings)
	}
}
