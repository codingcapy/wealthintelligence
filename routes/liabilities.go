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

func LiabilitiesRouter() chi.Router {
	r := chi.NewRouter()

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
		liabilities := []models.Liability{}
		err = db.DB.Select(&liabilities, "SELECT * FROM liabilities WHERE plan_id = $1", planID)
		if err != nil {
			log.Println("Error fetching liabilities:", err)
			http.Error(w, "Error fetching liabilities", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"liabilities": liabilities})
	})

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var body struct {
			PlanID   int    `json:"planId"`
			Name     string `json:"name"`
			Amount   int64  `json:"amount"`
			Interest int    `json:"interest"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var newLiability models.Liability
		err := db.DB.QueryRowx(
			`INSERT INTO liabilities (plan_id, name, amount, interest) VALUES ($1, $2, $3, $4) RETURNING *`,
			body.PlanID, body.Name, body.Amount, body.Interest,
		).StructScan(&newLiability)
		if err != nil {
			log.Println("Error creating liability:", err)
			http.Error(w, "Error creating liability", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"liability": newLiability})
	})

	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var body struct {
			LiabilityID int `json:"liabilityId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		var check []struct {
			LiabilityID int `db:"liability_id"`
		}
		err := db.DB.Select(&check,
			`SELECT a.liability_id FROM liabilities a
			 INNER JOIN plans p ON a.plan_id = p.plan_id
			 WHERE a.liability_id = $1 AND p."user" = $2`,
			body.LiabilityID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var deletedLiability models.Liability
		err = db.DB.QueryRowx(
			`DELETE FROM liabilities WHERE liabilities_id = $1 RETURNING *`,
			body.LiabilityID,
		).StructScan(&deletedLiability)
		if err != nil {
			log.Println("Error deleting liability:", err)
			http.Error(w, "Error deleting liability", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"liability": deletedLiability})
	})

	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var body struct {
			LiabilityID int    `json:"liabilityId"`
			Name        string `json:"name"`
			Amount      int64  `json:"amount"`
			Interest    int    `json:"interest"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		var check []struct {
			LiabilityID int `db:"liability_id"`
		}
		err := db.DB.Select(&check,
			`SELECT a.liability_id FROM liabilities a
			 INNER JOIN plans p ON a.plan_id = p.plan_id
			 WHERE a.liability_id = $1 AND p."user" = $2`,
			body.LiabilityID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var updatedLiability models.Liability
		err = db.DB.QueryRowx(
			`UPDATE liabilities SET name = $1, value = $2, roi = $3 WHERE liability_id = $4 RETURNING *`,
			body.Name, body.Amount, body.Interest, body.LiabilityID,
		).StructScan(&updatedLiability)
		if err != nil {
			log.Println("Error updating liability:", err)
			http.Error(w, "Error updating liability", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"liability": updatedLiability})
	})

	return r
}
