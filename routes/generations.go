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

func GenerationsRouter() chi.Router {
	r := chi.NewRouter()

	// GET /:planId — get generations for a plan
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

		generations := []models.Generation{}
		err = db.DB.Select(&generations, "SELECT * FROM generations WHERE plan_id = $1", planID)
		if err != nil {
			log.Println("Error fetching generations:", err)
			http.Error(w, "Error fetching generations", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"generations": generations})
	})

	// POST /delete — delete generation
	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			GenerationID int `json:"generationId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var check []struct {
			GenerationID int `db:"generation_id"`
		}
		err := db.DB.Select(&check,
			`SELECT g.generation_id FROM generations g
			 INNER JOIN plans p ON g.plan_id = p.plan_id
			 WHERE g.generation_id = $1 AND p."user" = $2`,
			body.GenerationID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var deletedGeneration models.Generation
		err = db.DB.QueryRowx(
			`DELETE FROM generations WHERE generation_id = $1 RETURNING *`,
			body.GenerationID,
		).StructScan(&deletedGeneration)
		if err != nil {
			log.Println("Error deleting generation:", err)
			http.Error(w, "Error deleting generation", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"generation": deletedGeneration})
	})

	return r
}
