package handlers

import (
	"net/http"

	"github.com/adamciksimon/public-api/billing"
	"github.com/adamciksimon/public-api/billing/services"
)

type Config struct {
	Store         *billing.PgStore
	ReturnURL     string
	SuccessURL    string
	CancelURL     string
	WebhookSecret string
}

func Register(mux *http.ServeMux, cfg Config) {
	plans := &PlanHandler{}
	mux.HandleFunc("GET /plans", plans.List)

	svc := &services.SubscriptionService{
		Store:      cfg.Store,
		SuccessURL: cfg.SuccessURL,
		CancelURL:  cfg.CancelURL,
		ReturnURL:  cfg.ReturnURL,
	}

	sub := &SubscriptionHandler{Service: svc}
	mux.HandleFunc("POST /subscription/checkout", sub.Checkout)
	mux.HandleFunc("POST /subscription/cancel", sub.Cancel)
	mux.HandleFunc("POST /subscription/portal", sub.Portal)

	wh := &WebhookHandler{Service: svc, Secret: cfg.WebhookSecret}
	mux.HandleFunc("POST /webhook", wh.Handle)
}
