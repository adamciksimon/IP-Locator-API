package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/adamciksimon/public-api/billing"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/billingportal/session"
)

type PortalHandler struct {
	Store     *billing.PgStore
	ReturnURL string
}

type createPortalRequest struct {
	CustomerEmail string `json:"customerEmail"`
}

func (h *PortalHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req createPortalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CustomerEmail == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	profile, err := h.Store.GetBillingProfileByEmail(r.Context(), req.CustomerEmail)
	if errors.Is(err, billing.ErrNotFound) {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("portal: get billing profile %s: %v", req.CustomerEmail, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if profile.ExternalID == "" {
		http.Error(w, "no billing account found", http.StatusUnprocessableEntity)
		return
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(profile.ExternalID),
		ReturnURL: stripe.String(h.ReturnURL),
	}

	ps, err := session.New(params)
	if err != nil {
		log.Printf("portal: create session %s: %v", req.CustomerEmail, err)
		http.Error(w, "failed to create portal session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": ps.URL})
}
