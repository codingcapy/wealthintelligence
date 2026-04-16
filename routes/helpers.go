package routes

import (
	"fmt"
	"os"

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
