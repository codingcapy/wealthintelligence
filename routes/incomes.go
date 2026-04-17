package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func IncomesRouter() chi.Router {
	r := chi.NewRouter()

	// GET /:planId — get incomes for a plan
	r.Get("/{planId}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		planID, err := strconv.Atoi(chi.URLParam(r, "planId"))
		if err != nil {
			http.Error(w, "Invalid plan ID", http.StatusBadRequest)
			return
		}

		// Verify plan belongs to user
		if err := ownershipCheck(userID, planID); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		incomes := []models.Income{}
		err = db.DB.Select(&incomes, "SELECT * FROM incomes WHERE plan_id = $1", planID)
		if err != nil {
			log.Println("Error fetching incomes:", err)
			http.Error(w, "Error fetching incomes", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"incomes": incomes})
	})

	// POST / — create income
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID   int    `json:"planId"`
			Company  string `json:"company"`
			Position string `json:"position"`
			Amount   int64  `json:"amount"`
			Tax      int    `json:"tax"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var newIncome models.Income
		err := db.DB.QueryRowx(
			`INSERT INTO incomes (plan_id, company, position, amount, tax) VALUES ($1, $2, $3, $4, $5) RETURNING *`,
			body.PlanID, body.Company, body.Position, body.Amount, body.Tax,
		).StructScan(&newIncome)
		if err != nil {
			log.Println("Error creating income:", err)
			http.Error(w, "Error creating income", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"income": newIncome})
	})

	// POST /delete — delete income
	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			IncomeID int `json:"incomeId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ownership check via join
		var check []struct {
			IncomeID int `db:"income_id"`
		}
		err := db.DB.Select(&check,
			`SELECT i.income_id FROM incomes i
			 INNER JOIN plans p ON i.plan_id = p.plan_id
			 WHERE i.income_id = $1 AND p."user" = $2`,
			body.IncomeID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var deletedIncome models.Income
		err = db.DB.QueryRowx(
			`DELETE FROM incomes WHERE income_id = $1 RETURNING *`,
			body.IncomeID,
		).StructScan(&deletedIncome)
		if err != nil {
			log.Println("Error deleting income:", err)
			http.Error(w, "Error deleting income", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"income": deletedIncome})
	})

	// POST /update — update income
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			IncomeID int    `json:"incomeId"`
			Company  string `json:"company"`
			Position string `json:"position"`
			Amount   int64  `json:"amount"`
			Tax      int    `json:"tax"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ownership check via join
		var check []struct {
			IncomeID int `db:"income_id"`
		}
		err := db.DB.Select(&check,
			`SELECT i.income_id FROM incomes i
			 INNER JOIN plans p ON i.plan_id = p.plan_id
			 WHERE i.income_id = $1 AND p."user" = $2`,
			body.IncomeID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var updatedIncome models.Income
		err = db.DB.QueryRowx(
			`UPDATE incomes SET company = $1, position = $2, amount = $3, tax = $4 WHERE income_id = $5 RETURNING *`,
			body.Company, body.Position, body.Amount, body.Tax, body.IncomeID,
		).StructScan(&updatedIncome)
		if err != nil {
			log.Println("Error updating income:", err)
			http.Error(w, "Error updating income", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"income": updatedIncome})
	})

	return r
}
