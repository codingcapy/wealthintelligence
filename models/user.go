package models

import "time"

type User struct {
	UserID      string    `json:"userId"      db:"user_id"`
	Username    string    `json:"username"    db:"username"`
	Email       string    `json:"email"       db:"email"`
	Password    string    `json:"password"    db:"password"`
	ProfilePic  *string   `json:"profilePic"  db:"profile_pic"`
	Role        string    `json:"role"        db:"role"`
	Status      string    `json:"status"      db:"status"`
	Preference  string    `json:"preference"  db:"preference"`
	CreatedAt   time.Time `json:"createdAt"   db:"created_at"`
	CurrentPlan int       `json:"currentPlan" db:"current_plan"`
}
