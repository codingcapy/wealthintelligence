package models

import "time"

type Expenditure struct {
	ExpenditureID int       `json:"expenditureId" db:"expenditure_id"`
	PlanID        int       `json:"planId"        db:"plan_id"`
	Name          string    `json:"name"          db:"name"`
	Amount        int64     `json:"amount"        db:"amount"`
	Status        string    `json:"status"        db:"status"`
	CreatedAt     time.Time `json:"createdAt"     db:"created_at"`
}
