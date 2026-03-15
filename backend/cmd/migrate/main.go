package main

import (
	"expense-tracker/backend/config"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.Envs

	psqlInfo := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBPublicHost, cfg.DBPort, cfg.DBName)

	migrationsPath, err := findMigrationsPath()
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.New(
		"file://"+migrationsPath,
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
		return
	}
	if cmd == "down" {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		return
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
		return
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
		return
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
		return
	}

	log.Fatalf("unknown subcommand %q", cmd)
}

func findMigrationsPath() (string, error) {
	candidates := []string{
		"backend/cmd/migrate/migrations",
		"cmd/migrate/migrations",
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				return "", err
			}
			return absPath, nil
		}
	}

	return "", fmt.Errorf("migration directory not found; checked %v", candidates)
}
