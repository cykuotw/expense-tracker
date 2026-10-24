package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMigration000043ContractsOnlyEmptyHistoricalTables(t *testing.T) {
	db := openMigrationTestDB(t)
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok)
	directory := filepath.Join(filepath.Dir(source), "migrations")
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(directory, name))
		require.NoError(t, err)
		return string(data)
	}
	historical := read("000041_add_ocr_improvement_evidence.up.sql")
	contract := read("000043_drop_empty_ocr_improvement_tables.up.sql")
	down := read("000043_drop_empty_ocr_improvement_tables.down.sql")
	for _, scenario := range []string{"empty", "consent", "evidence", "unexpected dependency"} {
		t.Run(scenario, func(t *testing.T) {
			tx, err := db.BeginTx(t.Context(), nil)
			require.NoError(t, err)
			defer tx.Rollback()
			schema := fmt.Sprintf("ocr_contract_%d", time.Now().UnixNano())
			_, err = tx.ExecContext(t.Context(), fmt.Sprintf(`
                CREATE SCHEMA %s;
                SET LOCAL search_path TO %s, pg_catalog;
                SET LOCAL lock_timeout TO '5s';
                CREATE TABLE users(id UUID PRIMARY KEY);
                CREATE TABLE expense(id UUID PRIMARY KEY, is_deleted BOOLEAN NOT NULL DEFAULT FALSE);
                CREATE TABLE expense_receipt(id UUID PRIMARY KEY, expense_id UUID NOT NULL REFERENCES expense(id), status TEXT NOT NULL);
                CREATE TABLE ocr_usage(request_id UUID PRIMARY KEY, input_tokens BIGINT NOT NULL);
                INSERT INTO users VALUES('00000000-0000-4000-8000-000000000001');
                INSERT INTO expense VALUES('00000000-0000-4000-8000-000000000002', FALSE);
                INSERT INTO expense_receipt VALUES('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002','finalized');
                INSERT INTO ocr_usage VALUES('00000000-0000-4000-8000-000000000004',123);
            `, schema, schema))
			require.NoError(t, err)
			require.NoError(t, executeSQLScript(tx, historical))
			switch scenario {
			case "consent":
				_, err = tx.Exec(`INSERT INTO ocr_improvement_consent(account_id,policy_version) VALUES('00000000-0000-4000-8000-000000000001','test')`)
			case "evidence":
				_, err = tx.Exec(`INSERT INTO ocr_improvement_evidence(id,account_id,run_id,consent_revision,policy_version,state)
                    VALUES('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000001',
                    '00000000-0000-4000-8000-000000000006',1,'test','pending')`)
			case "unexpected dependency":
				_, err = tx.Exec(`CREATE VIEW historical_consent_dependency AS SELECT account_id FROM ocr_improvement_consent`)
			}
			require.NoError(t, err)
			_, err = tx.Exec("SAVEPOINT contract_attempt")
			require.NoError(t, err)
			err = executeSQLScript(tx, contract)
			if scenario != "empty" {
				require.Error(t, err)
				if scenario == "consent" || scenario == "evidence" {
					require.ErrorContains(t, err, "not empty")
				}
				_, err = tx.Exec("ROLLBACK TO SAVEPOINT contract_attempt")
				require.NoError(t, err)
			} else {
				require.NoError(t, err)
				_, err = tx.Exec("SAVEPOINT down_attempt")
				require.NoError(t, err)
				require.ErrorContains(t, executeSQLScript(tx, down), "manual schema recovery")
				_, err = tx.Exec("ROLLBACK TO SAVEPOINT down_attempt")
				require.NoError(t, err)
			}
			var tablesExist, functionExists bool
			require.NoError(t, tx.QueryRow(`SELECT to_regclass('ocr_improvement_consent') IS NOT NULL
                AND to_regclass('ocr_improvement_evidence') IS NOT NULL,
                to_regprocedure('mark_ocr_evidence_for_cleanup()') IS NOT NULL`).Scan(&tablesExist, &functionExists))
			require.Equal(t, scenario != "empty", tablesExist)
			require.Equal(t, scenario != "empty", functionExists)
			var triggers int
			require.NoError(t, tx.QueryRow(`SELECT count(*) FROM pg_trigger WHERE NOT tgisinternal
                AND tgrelid IN ('expense'::regclass,'expense_receipt'::regclass)`).Scan(&triggers))
			if scenario == "empty" {
				require.Zero(t, triggers)
			} else {
				require.Equal(t, 2, triggers)
			}
			_, err = tx.Exec(`UPDATE expense SET is_deleted=TRUE; UPDATE expense_receipt SET status='deleting'`)
			require.NoError(t, err, "expense and receipt updates still work")
			var preserved bool
			require.NoError(t, tx.QueryRow(`SELECT (SELECT count(*) FROM expense)=1
                AND (SELECT count(*) FROM expense_receipt)=1 AND (SELECT input_tokens FROM ocr_usage)=123`).Scan(&preserved))
			require.True(t, preserved)
		})
	}
}
