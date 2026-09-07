package main

import (
	"context"
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	"expense-tracker/backend/services/notification"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	configuration := config.Envs
	database, err := db.NewPostgreSQLStorage(configuration)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	privateKey := strings.TrimSpace(os.Getenv("WEB_PUSH_VAPID_PRIVATE_KEY"))
	subject := strings.TrimSpace(os.Getenv("WEB_PUSH_VAPID_SUBJECT"))
	sender, err := notification.NewSender(notification.NewStore(database), configuration.WebPushVAPIDPublicKey, privateKey, subject)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	if err := sender.RunOnce(ctx); err != nil {
		log.Fatal(err)
	}
}
