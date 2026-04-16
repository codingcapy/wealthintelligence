package routes

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/scrypt"
)

func verifyPassword(hash, password string) (bool, error) {
	parts := strings.Split(hash, ":")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid hash format")
	}
	salt := parts[0]
	keyHex := parts[1]

	derivedKey, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 64)
	if err != nil {
		return false, err
	}

	expected, err := hex.DecodeString(keyHex)
	if err != nil {
		return false, err
	}

	return subtle.ConstantTimeCompare(derivedKey, expected) == 1, nil
}

func hashPassword(password string) (string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes)

	derivedKey, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 64)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", salt, hex.EncodeToString(derivedKey)), nil
}

func generateToken(userID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userID,
		"exp": time.Now().Add(14 * 24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secret))
}

func UserRouter() chi.Router {
	r := chi.NewRouter()

	// POST /api/v0/user/login
	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var user models.User
		err := db.DB.Get(&user, "SELECT * FROM users WHERE email = $1", body.Email)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"user": nil, "token": nil}})
			return
		}

		valid, err := verifyPassword(user.Password, body.Password)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"user": nil, "token": nil}})
			return
		}
		if !valid {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"user": nil, "token": nil}})
			return
		}

		token, err := generateToken(user.UserID)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"user": user, "token": token}})
	})

	r.Post("/validation", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Header does not exist", http.StatusForbidden)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			http.Error(w, "Invalid authorization header", http.StatusForbidden)
			return
		}
		tokenStr := parts[1]

		userID, err := extractUserIDFromToken(tokenStr)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var user models.User
		err = db.DB.Get(&user, "SELECT * FROM users WHERE user_id = $1", userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"user": user, "token": tokenStr}})
	})

	return r
}
