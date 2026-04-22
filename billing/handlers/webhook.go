package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/adamciksimon/public-api/billing/services"
	"github.com/stripe/stripe-go/v82/webhook"
)

type WebhookHandler struct {
	Service *services.SubscriptionService
	Secret  string
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), h.Secret)
	if err != nil {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if err := h.Service.HandleEvent(r.Context(), event); err != nil {
		log.Printf("webhook: handle %s: %v", event.Type, err)
	}

	w.WriteHeader(http.StatusOK)
}
