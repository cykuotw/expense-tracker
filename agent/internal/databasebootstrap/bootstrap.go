// Package databasebootstrap prepares the Agent schema and restricted runtime role.
package databasebootstrap

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"strings"
	"time"
)

// Run applies bundled migrations before provisioning the runtime role.
// Callers supply an operator connection and a filesystem containing migrations/.
func Run(ctx context.Context, db *sql.DB, files fs.FS, user, password string) error {
	if strings.TrimSpace(user) == "" || password == "" {
		return errors.New("Agent runtime role configuration is required")
	}
	if err := migrateUp(ctx, db, files); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return ProvisionRole(ctx, db, user, password)
}
