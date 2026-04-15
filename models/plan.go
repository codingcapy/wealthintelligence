package models

import "time"

type Plan struct {
	PlanID      int       `json:"planId"      db:"plan_id"`
	UserID      string    `json:"userId"      db:"user"`
	Title       string    `json:"title"       db:"title"`
	Icon        *string   `json:"icon"        db:"icon"`
	Currency    string    `json:"currency"    db:"currency"`
	Location    string    `json:"location"    db:"location"`
	YearOfBirth string    `json:"yearOfBirth" db:"year_of_birth"`
	Status      string    `json:"status"      db:"status"`
	CreatedAt   time.Time `json:"createdAt"   db:"created_at"`
}
