package ocrusage

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestStoreMigrationDeduplicationNullUsageAndAccountDeletion(t *testing.T) {
	dsn := os.Getenv("OCR_USAGE_TEST_DSN")
	if dsn == "" {
		t.Skip("set OCR_USAGE_TEST_DSN to an isolated PostgreSQL test database")
	}
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.PingContext(t.Context()))
	schema := fmt.Sprintf("ocr_usage_test_%d", time.Now().UnixNano())
	_, err = db.ExecContext(t.Context(), "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	isolated, err := sql.Open("pgx", parsed.String())
	require.NoError(t, err)
	defer isolated.Close()
	_, err = isolated.ExecContext(t.Context(), "CREATE TABLE users(id UUID PRIMARY KEY)")
	require.NoError(t, err)
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrations := filepath.Join(filepath.Dir(source), "../../cmd/migrate/migrations")
	up, err := os.ReadFile(filepath.Join(migrations, "000042_add_ocr_usage.up.sql"))
	require.NoError(t, err)
	_, err = isolated.ExecContext(t.Context(), string(up))
	require.NoError(t, err)
	r := sampleRecord()
	_, err = isolated.ExecContext(t.Context(), "INSERT INTO users(id) VALUES($1)", r.AccountID)
	require.NoError(t, err)
	store := NewStore(isolated)
	require.NoError(t, store.Save(t.Context(), r))
	r.Tokens = &Tokens{Input: 1, Total: 1}
	require.NoError(t, store.Save(t.Context(), r))
	var count, input int64
	require.NoError(t, isolated.QueryRow("SELECT count(*),max(input_tokens) FROM ocr_usage").Scan(&count, &input))
	require.Equal(t, int64(1), count)
	require.Equal(t, int64(100), input, "a retry must not overwrite accounting")
	r.RequestID = "3e33f107-5d2d-43d2-bc13-25bd84c3aef7"
	r.Tokens = nil
	r.UsageStatus = "missing"
	r.CostStatus = "usage_unavailable"
	r.EstimatedCostUSD = nil
	r.PricingVersion = ""
	r.Outcome = "request_failed"
	require.NoError(t, store.Save(t.Context(), r))
	var missing bool
	require.NoError(t, isolated.QueryRow("SELECT input_tokens IS NULL AND estimated_cost_usd IS NULL FROM ocr_usage WHERE request_id=$1", r.RequestID).Scan(&missing))
	require.True(t, missing)
	_, err = isolated.Exec("UPDATE ocr_usage SET output_tokens=-1 WHERE usage_status='available'")
	require.Error(t, err)
	_, err = isolated.Exec("DELETE FROM users WHERE id=$1", r.AccountID)
	require.NoError(t, err)
	require.NoError(t, isolated.QueryRow("SELECT count(*) FROM ocr_usage WHERE account_id IS NULL").Scan(&count))
	require.Equal(t, int64(2), count)
	down, err := os.ReadFile(filepath.Join(migrations, "000042_add_ocr_usage.down.sql"))
	require.NoError(t, err)
	_, err = isolated.Exec(string(down))
	require.NoError(t, err)
	var exists bool
	require.NoError(t, isolated.QueryRow("SELECT to_regclass('ocr_usage') IS NOT NULL").Scan(&exists))
	require.False(t, exists)
}
