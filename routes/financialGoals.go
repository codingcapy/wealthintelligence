package routes

import (
	"encoding/json"
	"net/http"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func FinancialGoalsRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		financialGoals := []models.FinancialGoal{}

		err := db.DB.Select(&financialGoals, "SELECT * FROM financial_goals")
		if err != nil {
			http.Error(w, "Failed to fetch financial goals", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(financialGoals)
	})

	return r
}
