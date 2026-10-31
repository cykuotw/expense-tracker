package databasebootstrap

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

var roleName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// ProvisionRole is operator-only local tooling, never called at request startup.
// The migrations must already exist. Password values are never returned/logged.
func ProvisionRole(ctx context.Context, admin *sql.DB, name, password string) error {
	if !roleName.MatchString(name) || !strings.HasPrefix(name, "agent_state_") || password == "" || strings.ContainsRune(password, 0) {
		return errors.New("invalid agent role configuration")
	}
	tx, err := admin.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("agent role provisioning failed")
	}
	defer tx.Rollback()
	var current string
	if tx.QueryRowContext(ctx, `SELECT current_user`).Scan(&current) != nil || current == name {
		return errors.New("agent role must differ from operator role")
	}
	var exists bool
	if tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)`, name).Scan(&exists) != nil {
		return errors.New("agent role provisioning failed")
	}
	quoted := pgx.Identifier{name}.Sanitize()
	var database, quotedPassword string
	if tx.QueryRowContext(ctx, `SELECT current_database(),quote_literal($1)`, password).Scan(&database, &quotedPassword) != nil {
		return errors.New("agent role provisioning failed")
	}
	if !exists {
		if _, err = tx.ExecContext(ctx, "CREATE ROLE "+quoted+" LOGIN"); err != nil {
			return errors.New("agent role provisioning failed")
		}
	} else {
		var privileged bool
		if tx.QueryRowContext(ctx, `SELECT rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls OR EXISTS(SELECT 1 FROM pg_tables WHERE schemaname='public' AND has_table_privilege($1,format('%I.%I',schemaname,tablename),'SELECT,INSERT,UPDATE,DELETE')) FROM pg_roles WHERE rolname=$1`, name).Scan(&privileged) != nil || privileged {
			return errors.New("refusing to repurpose an existing privileged role")
		}
	}
	var member bool
	if tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=(SELECT oid FROM pg_roles WHERE rolname=$1))`, name).Scan(&member) != nil || member {
		return errors.New("agent role must have no role memberships")
	}
	statements := []string{
		"ALTER ROLE " + quoted + " WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS PASSWORD " + quotedPassword,
		"ALTER ROLE " + quoted + " SET search_path = expense_agent, pg_catalog",
		"ALTER ROLE " + quoted + " SET statement_timeout = '3s'",
		"REVOKE ALL ON SCHEMA public FROM " + quoted,
		"REVOKE ALL ON ALL TABLES IN SCHEMA public FROM " + quoted,
		"GRANT CONNECT ON DATABASE " + pgx.Identifier{database}.Sanitize() + " TO " + quoted,
		"GRANT USAGE ON SCHEMA expense_agent TO " + quoted,
		"GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA expense_agent TO " + quoted,
	}
	for _, statement := range statements {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return errors.New("agent role provisioning failed")
		}
	}
	if tx.Commit() != nil {
		return errors.New("agent role provisioning failed")
	}
	return nil
}
