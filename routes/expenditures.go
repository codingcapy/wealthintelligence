package routes

import (
	"encoding/json"
	"net/http"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func ExpendituresRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		expenditures := []models.Expenditure{}

		err := db.DB.Select(&expenditures, "SELECT * FROM expenditures")
		if err != nil {
			http.Error(w, "Failed to fetch expenditures", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expenditures)
	})

	return r
}
