package routes

import (
	"fmt"
	"net/http"
	"os"

	"wealthintelligence/db"

	"github.com/golang-jwt/jwt/v5"
)

func extractUserIDFromToken(tokenStr string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	claims := token.Claims.(jwt.MapClaims)
	id, ok := claims["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}
	return id, nil
}

// countLimitCheck returns an error if the given table already has 20 or more rows for the plan.
func countLimitCheck(w http.ResponseWriter, table string, planID int) bool {
	var count int
	err := db.DB.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE plan_id = $1", table), planID,
	).Scan(&count)
	if err != nil {
		http.Error(w, "Error checking limit", http.StatusInternalServerError)
		return false
	}
	if count >= 20 {
		http.Error(w, "Limit of 20 entries per plan reached", http.StatusUnprocessableEntity)
		return false
	}
	return true
}
