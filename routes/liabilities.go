package routes

import (
	"encoding/json"
	"net/http"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func LiabilitiesRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		liabilities := []models.Liability{}

		err := db.DB.Select(&liabilities, "SELECT * FROM liabilities")
		if err != nil {
			http.Error(w, "Failed to fetch liabilities", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(liabilities)
	})

	return r
}
