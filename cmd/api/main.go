package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v82"

	"github.com/adamciksimon/public-api/billing"
	"github.com/adamciksimon/public-api/billing/services"
	"github.com/adamciksimon/public-api/database"
	"github.com/adamciksimon/public-api/router"
)

func main() {
	reset := flag.Bool("reset", false, "drop and recreate all tables")
	syncPlans := flag.Bool("sync-plans", false, "create missing Stripe products for plans")
	flag.Parse()

	_ = godotenv.Load()

	stripe.Key = getenv("STRIPE_SECRET_KEY", "")

	db, err := database.New(database.DbConfig{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     getenv("DB_PORT", "5432"),
		User:     getenv("DB_USER", "postgres"),
		Password: getenv("DB_PASSWORD", ""),
		DBName:   getenv("DB_NAME", "public_api"),
		SSLMode:  getenv("DB_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	if *reset {
		if err := database.Reset(context.Background(), db); err != nil {
			log.Fatalf("reset: %v", err)
		}
	} else if err := database.Migrate(context.Background(), db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := billing.Seed(context.Background(), db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	s := billing.NewStore(db)

	if *syncPlans {
		svc := &services.SubscriptionService{Store: s}
		if err := svc.SyncPlans(context.Background()); err != nil {
			log.Fatalf("sync plans: %v", err)
		}
		return
	}

	frontendURL := getenv("FRONTEND_URL", "http://localhost:3000")
	server := http.Server{
		Addr: ":3333",
		Handler: router.New(s, router.Config{
			ReturnURL:     frontendURL + "/subscription",
			SuccessURL:    frontendURL + "/subscription?status=success",
			CancelURL:     frontendURL + "/subscription?status=canceled",
			WebhookSecret: getenv("STRIPE_WEBHOOK_SECRET", ""),
		}),
	}

	log.Println("Starting server on :3333")
	log.Fatal(server.ListenAndServe())
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
