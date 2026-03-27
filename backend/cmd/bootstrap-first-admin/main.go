package main

import (
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/services/user"
	"log"
	"os"
)

func requireEnv(name string) string {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		log.Fatalf("missing required environment variable %s", name)
	}
	return value
}

func main() {
	storage, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()

	if err := storage.Ping(); err != nil {
		log.Fatal(err)
	}

	created, err := user.BootstrapFirstAdmin(user.NewStore(storage), user.FirstAdminInput{
		Email:     requireEnv("FIRST_ADMIN_EMAIL"),
		Password:  requireEnv("FIRST_ADMIN_PASSWORD"),
		Firstname: requireEnv("FIRST_ADMIN_FIRSTNAME"),
		Lastname:  requireEnv("FIRST_ADMIN_LASTNAME"),
		Nickname:  os.Getenv("FIRST_ADMIN_NICKNAME"),
	}, user.BootstrapDeps{})
	if err != nil {
		log.Fatal(err)
	}

	if created {
		log.Println("first admin user created")
		return
	}

	log.Println("first admin user bootstrap skipped because an admin already exists")
}
