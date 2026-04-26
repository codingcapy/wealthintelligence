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

	// GET /:planId — get generations for a plan (cursor-based pagination)
	// Query params: cursor=<generation_id> (exclusive, for next page), limit=<n> (default 20)
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

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		cursor := 0
		if c := r.URL.Query().Get("cursor"); c != "" {
			if parsed, err := strconv.Atoi(c); err == nil {
				cursor = parsed
			}
		}

		generations := []models.Generation{}
		if cursor > 0 {
			err = db.DB.Select(&generations,
				"SELECT * FROM generations WHERE plan_id = $1 AND generation_id < $2 ORDER BY generation_id DESC LIMIT $3",
				planID, cursor, limit+1,
			)
		} else {
			err = db.DB.Select(&generations,
				"SELECT * FROM generations WHERE plan_id = $1 ORDER BY generation_id DESC LIMIT $2",
				planID, limit+1,
			)
		}
		if err != nil {
			log.Println("Error fetching generations:", err)
			http.Error(w, "Error fetching generations", http.StatusInternalServerError)
			return
		}

		var nextCursor *int
		if len(generations) > limit {
			generations = generations[:limit]
			id := generations[limit-1].GenerationID
			nextCursor = &id
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"generations": generations, "nextCursor": nextCursor})
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
