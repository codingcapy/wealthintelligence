package models

import "time"

type Generation struct {
	GenerationID int       `json:"generationId" db:"generation_id"`
	PlanID       int       `json:"planId"       db:"plan_id"`
	Content      string    `json:"content"      db:"content"`
	CreatedAt    time.Time `json:"createdAt"    db:"created_at"`
}
