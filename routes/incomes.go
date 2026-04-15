package routes

import (
	"encoding/json"
	"net/http"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func IncomesRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		incomes := []models.Income{}

		err := db.DB.Select(&incomes, "SELECT * FROM incomes")
		if err != nil {
			http.Error(w, "Failed to fetch incomes", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(incomes)
	})

	return r
}
