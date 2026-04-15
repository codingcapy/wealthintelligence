package routes

import (
	"encoding/json"
	"net/http"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func GenerationsRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		generations := []models.Generation{}

		err := db.DB.Select(&generations, "SELECT * FROM generations")
		if err != nil {
			http.Error(w, "Failed to fetch generations", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generations)
	})

	return r
}
