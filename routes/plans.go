package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
)

func PlansRouter() chi.Router {
	r := chi.NewRouter()

	// GET / — get all plans for the authenticated user
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var plans []models.Plan
		err := db.DB.Select(&plans, `SELECT * FROM plans WHERE "user" = $1 ORDER BY created_at DESC`, userID)
		if err != nil {
			log.Println("Error fetching plans:", err)
			http.Error(w, "Error fetching plans", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"plans": plans})
	})

	// GET /:planId — get plan by id
	r.Get("/{planId}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		planId, err := strconv.Atoi(chi.URLParam(r, "planId"))
		if err != nil {
			http.Error(w, "Invalid plan ID", http.StatusBadRequest)
			return
		}

		var plan models.Plan
		err = db.DB.Get(&plan, "SELECT * FROM plans WHERE plan_id = $1 AND \"user\" = $2", planId, userID)
		if err != nil {
			log.Println("Error fetching plan:", err)
			http.Error(w, "Plan not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"plan": plan})
	})

	// POST / — create plan
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var planCount int
		err := db.DB.QueryRow(`SELECT COUNT(*) FROM plans WHERE "user" = $1`, userID).Scan(&planCount)
		if err != nil {
			http.Error(w, "Error checking limit", http.StatusInternalServerError)
			return
		}
		if planCount >= 20 {
			http.Error(w, "Limit of 20 plans reached", http.StatusUnprocessableEntity)
			return
		}

		var newPlan models.Plan
		err = db.DB.QueryRowx(
			`INSERT INTO plans ("user", title) VALUES ($1, $2) RETURNING *`,
			userID, body.Title,
		).StructScan(&newPlan)
		if err != nil {
			log.Println("Error creating plan:", err)
			http.Error(w, "Error creating plan", http.StatusInternalServerError)
			return
		}

		// Update user's current plan
		var updatedUser models.User
		err = db.DB.QueryRowx(
			`UPDATE users SET current_plan = $1 WHERE user_id = $2 RETURNING *`,
			newPlan.PlanID, userID,
		).StructScan(&updatedUser)
		if err != nil {
			log.Println("Error updating current plan:", err)
			http.Error(w, "Error updating current plan", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user": updatedUser})
	})

	// POST /delete — delete plan
	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID int `json:"planId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ownership check
		var ownershipCheck []models.Plan
		err := db.DB.Select(&ownershipCheck, `SELECT * FROM plans WHERE plan_id = $1 AND "user" = $2`, body.PlanID, userID)
		if err != nil {
			http.Error(w, "Ownership check failed", http.StatusInternalServerError)
			return
		}
		if len(ownershipCheck) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Delete plan
		_, err = db.DB.Exec(`DELETE FROM plans WHERE plan_id = $1`, body.PlanID)
		if err != nil {
			log.Println("Error deleting plan:", err)
			http.Error(w, "Error deleting plan", http.StatusInternalServerError)
			return
		}

		// Fetch user
		var user models.User
		err = db.DB.Get(&user, "SELECT * FROM users WHERE user_id = $1", userID)
		if err != nil {
			http.Error(w, "User query failed", http.StatusInternalServerError)
			return
		}

		// Check remaining plans
		var remainingPlans []models.Plan
		err = db.DB.Select(&remainingPlans, `SELECT * FROM plans WHERE "user" = $1`, userID)
		if err != nil {
			http.Error(w, "Plans check failed", http.StatusInternalServerError)
			return
		}

		var updatedUser models.User
		if len(remainingPlans) == 0 {
			// No plans left — auto-create one
			var newPlan models.Plan
			err = db.DB.QueryRowx(
				`INSERT INTO plans ("user", title) VALUES ($1, $2) RETURNING *`,
				userID, user.Username+"'s plan",
			).StructScan(&newPlan)
			if err != nil {
				log.Println("Error creating plan:", err)
				http.Error(w, "Error creating plan", http.StatusInternalServerError)
				return
			}
			err = db.DB.QueryRowx(
				`UPDATE users SET current_plan = $1 WHERE user_id = $2 RETURNING *`,
				newPlan.PlanID, userID,
			).StructScan(&updatedUser)
		} else {
			// Switch to first remaining plan
			err = db.DB.QueryRowx(
				`UPDATE users SET current_plan = $1 WHERE user_id = $2 RETURNING *`,
				remainingPlans[0].PlanID, userID,
			).StructScan(&updatedUser)
		}
		if err != nil {
			log.Println("Error updating current plan:", err)
			http.Error(w, "Error updating current plan", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user": updatedUser})
	})

	// POST /update — update title
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID int    `json:"planId"`
			Title  string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		var updatedPlan models.Plan
		err := db.DB.QueryRowx(
			`UPDATE plans SET title = $1 WHERE plan_id = $2 RETURNING *`,
			body.Title, body.PlanID,
		).StructScan(&updatedPlan)
		if err != nil {
			log.Println("Error updating plan title:", err)
			http.Error(w, "Error updating plan title", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"plan": updatedPlan})
	})

	// POST /update/currency
	r.Post("/update/currency", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID   int    `json:"planId"`
			Currency string `json:"currency"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		var updatedPlan models.Plan
		err := db.DB.QueryRowx(
			`UPDATE plans SET currency = $1 WHERE plan_id = $2 RETURNING *`,
			body.Currency, body.PlanID,
		).StructScan(&updatedPlan)
		if err != nil {
			log.Println("Error updating currency:", err)
			http.Error(w, "Error updating currency", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"plan": updatedPlan})
	})

	// POST /update/yearofbirth
	r.Post("/update/yearofbirth", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID      int    `json:"planId"`
			YearOfBirth string `json:"yearOfBirth"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		var updatedPlan models.Plan
		err := db.DB.QueryRowx(
			`UPDATE plans SET year_of_birth = $1 WHERE plan_id = $2 RETURNING *`,
			body.YearOfBirth, body.PlanID,
		).StructScan(&updatedPlan)
		if err != nil {
			log.Println("Error updating year of birth:", err)
			http.Error(w, "Error updating year of birth", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"plan": updatedPlan})
	})

	// POST /update/location
	r.Post("/update/location", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID   int    `json:"planId"`
			Location string `json:"location"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		var updatedPlan models.Plan
		err := db.DB.QueryRowx(
			`UPDATE plans SET location = $1 WHERE plan_id = $2 RETURNING *`,
			body.Location, body.PlanID,
		).StructScan(&updatedPlan)
		if err != nil {
			log.Println("Error updating location:", err)
			http.Error(w, "Error updating location", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"plan": updatedPlan})
	})

	return r
}

// ownershipCheck verifies a plan belongs to the user
func ownershipCheck(userID string, planID int) error {
	var plans []models.Plan
	err := db.DB.Select(&plans, `SELECT * FROM plans WHERE plan_id = $1 AND "user" = $2`, planID, userID)
	if err != nil {
		return fmt.Errorf("ownership check failed")
	}
	if len(plans) == 0 {
		return fmt.Errorf("unauthorized")
	}
	return nil
}
