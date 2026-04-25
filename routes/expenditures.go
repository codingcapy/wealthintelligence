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

func ExpendituresRouter() chi.Router {
	r := chi.NewRouter()

	// GET /:planId — get expenditures for a plan
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

		if err := ownershipCheck(userID, planID); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		expenditures := []models.Expenditure{}
		err = db.DB.Select(&expenditures, "SELECT * FROM expenditures WHERE plan_id = $1", planID)
		if err != nil {
			log.Println("Error fetching expenditures:", err)
			http.Error(w, "Error fetching expenditures", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"expenditures": expenditures})
	})

	// POST / — create expenditure
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID int    `json:"planId"`
			Name   string `json:"name"`
			Amount int64  `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !countLimitCheck(w, "expenditures", body.PlanID) {
			return
		}

		var newExpenditure models.Expenditure
		err := db.DB.QueryRowx(
			`INSERT INTO expenditures (plan_id, name, amount) VALUES ($1, $2, $3) RETURNING *`,
			body.PlanID, body.Name, body.Amount,
		).StructScan(&newExpenditure)
		if err != nil {
			log.Println("Error creating expenditure:", err)
			http.Error(w, "Error creating expenditure", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"expenditure": newExpenditure})
	})

	// POST /delete — delete expenditure
	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			ExpenditureID int `json:"expenditureId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var check []struct {
			ExpenditureID int `db:"expenditure_id"`
		}
		err := db.DB.Select(&check,
			`SELECT e.expenditure_id FROM expenditures e
			 INNER JOIN plans p ON e.plan_id = p.plan_id
			 WHERE e.expenditure_id = $1 AND p."user" = $2`,
			body.ExpenditureID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var deletedExpenditure models.Expenditure
		err = db.DB.QueryRowx(
			`DELETE FROM expenditures WHERE expenditure_id = $1 RETURNING *`,
			body.ExpenditureID,
		).StructScan(&deletedExpenditure)
		if err != nil {
			log.Println("Error deleting expenditure:", err)
			http.Error(w, "Error deleting expenditure", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"expenditure": deletedExpenditure})
	})

	// POST /update — update expenditure
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			ExpenditureID int    `json:"expenditureId"`
			Name          string `json:"name"`
			Amount        int64  `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var check []struct {
			ExpenditureID int `db:"expenditure_id"`
		}
		err := db.DB.Select(&check,
			`SELECT e.expenditure_id FROM expenditures e
			 INNER JOIN plans p ON e.plan_id = p.plan_id
			 WHERE e.expenditure_id = $1 AND p."user" = $2`,
			body.ExpenditureID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var updatedExpenditure models.Expenditure
		err = db.DB.QueryRowx(
			`UPDATE expenditures SET name = $1, amount = $2 WHERE expenditure_id = $3 RETURNING *`,
			body.Name, body.Amount, body.ExpenditureID,
		).StructScan(&updatedExpenditure)
		if err != nil {
			log.Println("Error updating expenditure:", err)
			http.Error(w, "Error updating expenditure", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"expenditure": updatedExpenditure})
	})

	return r
}
