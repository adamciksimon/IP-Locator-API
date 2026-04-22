package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/adamciksimon/public-api/billing"
)

type PlanHandler struct{}

func (h *PlanHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(billing.Plans)
}
