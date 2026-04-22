package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/adamciksimon/public-api/billing"
	"github.com/adamciksimon/public-api/billing/services"
)

type SubscriptionHandler struct {
	Service *services.SubscriptionService
}

type checkoutRequest struct {
	PlanUID       string `json:"planUid"`
	CustomerEmail string `json:"customerEmail"`
}

func (h *SubscriptionHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CustomerEmail == "" || req.PlanUID == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	result, err := h.Service.Checkout(r.Context(), req.CustomerEmail, req.PlanUID)
	if errors.Is(err, billing.ErrNotFound) {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("checkout: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type portalRequest struct {
	CustomerEmail string `json:"customerEmail"`
}

func (h *SubscriptionHandler) Portal(w http.ResponseWriter, r *http.Request) {
	var req portalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CustomerEmail == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	url, err := h.Service.Portal(r.Context(), req.CustomerEmail)
	if err != nil {
		log.Printf("portal: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

type cancelRequest struct {
	CustomerEmail string `json:"customerEmail"`
}

func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req cancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CustomerEmail == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	err := h.Service.Cancel(r.Context(), req.CustomerEmail)
	if errors.Is(err, billing.ErrNotFound) {
		http.Error(w, "subscription not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("cancel: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}
