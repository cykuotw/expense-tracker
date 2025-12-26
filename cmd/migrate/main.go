package main

import (
	"expense-tracker/config"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.Envs

	psqlInfo := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBPublicHost, cfg.DBPort, cfg.DBName)

	m, err := migrate.New(
		"file://cmd/migrate/migrations",
		psqlInfo,
	)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 {
		log.Fatal("expected 'up', 'down', 'step <n>', 'migrate <v>', or 'force <version>' subcommands")
	}

	cmd := os.Args[1]
	if cmd == "up" {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
	}
	if cmd == "down" {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
	}
	if cmd == "step" {
		if len(os.Args) < 3 {
			log.Fatal("step requires a number")
		}
		n, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Steps(n); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
	}
	if cmd == "migrate" {
		if len(os.Args) < 3 {
			log.Fatal("migrate requires a version number")
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Migrate(uint(v)); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
	}
	if cmd == "force" {
		if len(os.Args) < 3 {
			log.Fatal("force requires a version number")
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Force(v); err != nil {
			log.Fatal(err)
		}
	}
}
