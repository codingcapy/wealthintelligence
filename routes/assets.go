package routes

import (
	"encoding/json"
	"net/http"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func AssetsRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assets := []models.Asset{}

		err := db.DB.Select(&assets, "SELECT * FROM assets")
		if err != nil {
			http.Error(w, "Failed to fetch assets", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assets)
	})

	return r
}
