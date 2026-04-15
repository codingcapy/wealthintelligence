package models

import "time"

type Liability struct {
	LiabilityID int       `json:"liabilityId" db:"liability_id"`
	PlanID      int       `json:"planId"      db:"plan_id"`
	Name        string    `json:"name"        db:"name"`
	Amount      int64     `json:"amount"      db:"amount"`
	Interest    int       `json:"interest"    db:"interest"`
	Status      string    `json:"status"      db:"status"`
	CreatedAt   time.Time `json:"createdAt"   db:"created_at"`
}
