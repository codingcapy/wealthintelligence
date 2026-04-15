package models

import "time"

type FinancialGoal struct {
	FinancialGoalID int       `json:"financialGoalId" db:"financial_goal_id"`
	PlanID          int       `json:"planId"          db:"plan_id"`
	Name            string    `json:"name"            db:"name"`
	Amount          int64     `json:"amount"          db:"amount"`
	TargetDate      time.Time `json:"targetDate"      db:"target_date"`
	CreatedAt       time.Time `json:"createdAt"       db:"created_at"`
}
