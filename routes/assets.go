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

func AssetsRouter() chi.Router {
	r := chi.NewRouter()

	// GET /:planId — get assets for a plan
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

		assets := []models.Asset{}
		err = db.DB.Select(&assets, "SELECT * FROM assets WHERE plan_id = $1", planID)
		if err != nil {
			log.Println("Error fetching assets:", err)
			http.Error(w, "Error fetching assets", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"assets": assets})
	})

	// POST / — create asset
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			PlanID int    `json:"planId"`
			Name   string `json:"name"`
			Value  int64  `json:"value"`
			ROI    int    `json:"roi"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ownershipCheck(userID, body.PlanID); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var newAsset models.Asset
		err := db.DB.QueryRowx(
			`INSERT INTO assets (plan_id, name, value, roi) VALUES ($1, $2, $3, $4) RETURNING *`,
			body.PlanID, body.Name, body.Value, body.ROI,
		).StructScan(&newAsset)
		if err != nil {
			log.Println("Error creating asset:", err)
			http.Error(w, "Error creating asset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"asset": newAsset})
	})

	// POST /delete — delete asset
	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			AssetID int `json:"assetId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var check []struct {
			AssetID int `db:"asset_id"`
		}
		err := db.DB.Select(&check,
			`SELECT a.asset_id FROM assets a
			 INNER JOIN plans p ON a.plan_id = p.plan_id
			 WHERE a.asset_id = $1 AND p."user" = $2`,
			body.AssetID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var deletedAsset models.Asset
		err = db.DB.QueryRowx(
			`DELETE FROM assets WHERE asset_id = $1 RETURNING *`,
			body.AssetID,
		).StructScan(&deletedAsset)
		if err != nil {
			log.Println("Error deleting asset:", err)
			http.Error(w, "Error deleting asset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"asset": deletedAsset})
	})

	// POST /update — update asset
	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			AssetID int    `json:"assetId"`
			Name    string `json:"name"`
			Value   int64  `json:"value"`
			ROI     int    `json:"roi"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var check []struct {
			AssetID int `db:"asset_id"`
		}
		err := db.DB.Select(&check,
			`SELECT a.asset_id FROM assets a
			 INNER JOIN plans p ON a.plan_id = p.plan_id
			 WHERE a.asset_id = $1 AND p."user" = $2`,
			body.AssetID, userID,
		)
		if err != nil || len(check) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var updatedAsset models.Asset
		err = db.DB.QueryRowx(
			`UPDATE assets SET name = $1, value = $2, roi = $3 WHERE asset_id = $4 RETURNING *`,
			body.Name, body.Value, body.ROI, body.AssetID,
		).StructScan(&updatedAsset)
		if err != nil {
			log.Println("Error updating asset:", err)
			http.Error(w, "Error updating asset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"asset": updatedAsset})
	})

	return r
}
