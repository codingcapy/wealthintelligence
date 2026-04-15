package models

import "time"

type Asset struct {
	AssetID   int       `json:"assetId"   db:"asset_id"`
	PlanID    int       `json:"planId"    db:"plan_id"`
	Name      string    `json:"name"      db:"name"`
	Value     int64     `json:"value"     db:"value"`
	ROI       int       `json:"roi"       db:"roi"`
	Status    string    `json:"status"    db:"status"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}
