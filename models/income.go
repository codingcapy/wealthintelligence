package models

import "time"

type Income struct {
	IncomeID  int       `json:"incomeId"  db:"income_id"`
	PlanID    int       `json:"planId"    db:"plan_id"`
	Company   string    `json:"company"   db:"company"`
	Position  string    `json:"position"  db:"position"`
	Amount    int64     `json:"amount"    db:"amount"`
	Tax       int       `json:"tax"       db:"tax"`
	Status    string    `json:"status"    db:"status"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}
