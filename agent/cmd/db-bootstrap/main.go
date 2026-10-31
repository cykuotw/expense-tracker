// Command db-bootstrap applies Agent migrations and prepares the runtime role.
package main

import (
	"context"
	"database/sql"
	"embed"
	"log"
	"os"
	"strings"

	"expense-tracker/agent/internal/databasebootstrap"
	"expense-tracker/internal/localonly"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	if len(os.Args) != 1 {
		log.Fatal("usage: db-bootstrap")
	}
	if localonly.CheckRuntime() != nil {
		log.Fatal("Agent database bootstrap is local operator tooling only")
	}
	dsn := os.Getenv("AGENT_STATE_ADMIN_DATABASE_URL")
	if dsn == "" {
		log.Fatal("AGENT_STATE_ADMIN_DATABASE_URL is required")
	}
	user, password := os.Getenv("AGENT_STATE_DB_USER"), os.Getenv("AGENT_STATE_DB_PASSWORD")
	if strings.TrimSpace(user) == "" || password == "" {
		log.Fatal("AGENT_STATE_DB_USER and AGENT_STATE_DB_PASSWORD are required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("operator database unavailable")
	}
	defer db.Close()
	if err := databasebootstrap.Run(context.Background(), db, migrations, user, password); err != nil {
		// Driver errors can contain connection credentials or SQL payloads.
		log.Fatal("Agent database bootstrap failed; check connectivity, operator permissions, migration state, and runtime role configuration")
	}
	log.Print("Agent migrations and runtime role prepared")
}
