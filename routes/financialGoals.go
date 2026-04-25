package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func FinancialGoalsRouter() chi.Router {
	r := chi.NewRouter()

	// GET /:planId — get financial goals for a plan
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

		financialGoals := []models.FinancialGoal{}
		err = db.DB.Select(&financialGoals, "SELECT * FROM financial_goals WHERE plan_id = $1", planID)
		if err != nil {
			log.Println("Error fetching financial goals:", err)
			http.Error(w, "Error fetching financial goals", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"financialGoals": financialGoals})
	})

	// POST / — create financial goal
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID     int       `json:"planId"`
			Name       string    `json:"name"`
			Amount     int64     `json:"amount"`
			TargetDate time.Time `json:"targetDate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !countLimitCheck(w, "financial_goals", body.PlanID) {
			return
		}

		var newGoal models.FinancialGoal
		err := db.DB.QueryRowx(
			`INSERT INTO financial_goals (plan_id, name, amount, target_date) VALUES ($1, $2, $3, $4) RETURNING *`,
			body.PlanID, body.Name, body.Amount, body.TargetDate,
		).StructScan(&newGoal)
		if err != nil {
			log.Println("Error creating financial goal:", err)
			http.Error(w, "Error creating financial goal", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"financialGoal": newGoal})
	})

	// POST /delete — delete financial goal
	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			FinancialGoalID int `json:"financialGoalId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var check []struct {
			FinancialGoalID int `db:"financial_goal_id"`
		}
		err := db.DB.Select(&check,
			`SELECT fg.financial_goal_id FROM financial_goals fg
			 INNER JOIN plans p ON fg.plan_id = p.plan_id
			 WHERE fg.financial_goal_id = $1 AND p."user" = $2`,
			body.FinancialGoalID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var deletedGoal models.FinancialGoal
		err = db.DB.QueryRowx(
			`DELETE FROM financial_goals WHERE financial_goal_id = $1 RETURNING *`,
			body.FinancialGoalID,
		).StructScan(&deletedGoal)
		if err != nil {
			log.Println("Error deleting financial goal:", err)
			http.Error(w, "Error deleting financial goal", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"financialGoal": deletedGoal})
	})

	// POST /update — update financial goal
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			FinancialGoalID int       `json:"financialGoalId"`
			Name            string    `json:"name"`
			Amount          int64     `json:"amount"`
			TargetDate      time.Time `json:"targetDate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var check []struct {
			FinancialGoalID int `db:"financial_goal_id"`
		}
		err := db.DB.Select(&check,
			`SELECT fg.financial_goal_id FROM financial_goals fg
			 INNER JOIN plans p ON fg.plan_id = p.plan_id
			 WHERE fg.financial_goal_id = $1 AND p."user" = $2`,
			body.FinancialGoalID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var updatedGoal models.FinancialGoal
		err = db.DB.QueryRowx(
			`UPDATE financial_goals SET name = $1, amount = $2, target_date = $3 WHERE financial_goal_id = $4 RETURNING *`,
			body.Name, body.Amount, body.TargetDate, body.FinancialGoalID,
		).StructScan(&updatedGoal)
		if err != nil {
			log.Println("Error updating financial goal:", err)
			http.Error(w, "Error updating financial goal", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"financialGoal": updatedGoal})
	})

	return r
}
