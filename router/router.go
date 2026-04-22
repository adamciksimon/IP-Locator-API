package router

import (
	"net/http"

	"github.com/adamciksimon/public-api/billing"
	billinghandlers "github.com/adamciksimon/public-api/billing/handlers"
	"github.com/adamciksimon/public-api/middleware"
)

type Config struct {
	ReturnURL     string
	SuccessURL    string
	CancelURL     string
	WebhookSecret string
}

func New(s *billing.PgStore, cfg Config) http.Handler {
	mux := http.NewServeMux()

	billinghandlers.Register(mux, billinghandlers.Config{
		Store:         s,
		ReturnURL:     cfg.ReturnURL,
		SuccessURL:    cfg.SuccessURL,
		CancelURL:     cfg.CancelURL,
		WebhookSecret: cfg.WebhookSecret,
	})

	return middleware.ChainMiddleware(middleware.Logging)(mux)
}
