package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"wealthintelligence/db"
	"wealthintelligence/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	resend "github.com/resend/resend-go/v2"
)

func UsersRouter() chi.Router {
	r := chi.NewRouter()

	// POST / — create user + auto-create plan
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check email uniqueness
		var emailCheck []models.User
		err := db.DB.Select(&emailCheck, "SELECT * FROM users WHERE email = $1", body.Email)
		if err != nil {
			http.Error(w, "Error while fetching user", http.StatusInternalServerError)
			return
		}
		if len(emailCheck) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "An account with this email already exists"})
			return
		}

		// Check username uniqueness
		var usernameCheck []models.User
		err = db.DB.Select(&usernameCheck, "SELECT * FROM users WHERE username = $1", body.Username)
		if err != nil {
			http.Error(w, "Error while fetching user", http.StatusInternalServerError)
			return
		}
		if len(usernameCheck) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "An account with this username already exists"})
			return
		}

		// Hash password
		encrypted, err := hashPassword(body.Password)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}

		// Insert user
		userID := uuid.New().String()
		var newUser models.User
		err = db.DB.QueryRowx(
			`INSERT INTO users (user_id, username, email, password) VALUES ($1, $2, $3, $4) RETURNING *`,
			userID, body.Username, body.Email, encrypted,
		).StructScan(&newUser)
		if err != nil {
			http.Error(w, "Error while creating user", http.StatusInternalServerError)
			return
		}

		// Auto-create plan
		var planID int
		err = db.DB.QueryRow(
			`INSERT INTO plans ("user", title) VALUES ($1, $2) RETURNING plan_id`,
			newUser.UserID, newUser.Username+"'s plan",
		).Scan(&planID)
		if err != nil {
			http.Error(w, "Error while creating plan", http.StatusInternalServerError)
			return
		}

		// Update user's current_plan
		err = db.DB.QueryRowx(
			`UPDATE users SET current_plan = $1 WHERE user_id = $2 RETURNING *`,
			planID, newUser.UserID,
		).StructScan(&newUser)
		if err != nil {
			http.Error(w, "Error while updating current plan", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user": newUser})
	})

	// POST /update/currentplan
	r.Post("/update/currentplan", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			CurrentPlan int `json:"currentPlan"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var updatedUser models.User
		err := db.DB.QueryRowx(
			`UPDATE users SET current_plan = $1 WHERE user_id = $2 RETURNING *`,
			body.CurrentPlan, userID,
		).StructScan(&updatedUser)
		if err != nil {
			http.Error(w, "Error while updating current plan", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user": updatedUser})
	})

	// POST /update/password
	r.Post("/update/password", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		encrypted, err := hashPassword(body.Password)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}

		var updatedUser models.User
		err = db.DB.QueryRowx(
			`UPDATE users SET password = $1 WHERE user_id = $2 RETURNING *`,
			encrypted, userID,
		).StructScan(&updatedUser)
		if err != nil {
			http.Error(w, "Error while updating password", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user": updatedUser})
	})

	// POST /passwordreset
	r.Post("/passwordreset", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Generate temp password
		codeBytes := make([]byte, 32)
		rand.Read(codeBytes)
		code := hex.EncodeToString(codeBytes)
		hashedCode, err := hashPassword(code)
		if err != nil {
			http.Error(w, "Error hashing code", http.StatusInternalServerError)
			return
		}

		// Fetch user
		var user models.User
		err = db.DB.Get(&user, "SELECT * FROM users WHERE email = $1", body.Email)
		if err != nil {
			// silent — don't leak existence
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}

		originalPassword := user.Password

		// Update to temp password
		_, err = db.DB.Exec(
			`UPDATE users SET password = $1 WHERE email = $2`,
			hashedCode, body.Email,
		)
		if err != nil {
			http.Error(w, "Error while updating password", http.StatusInternalServerError)
			return
		}

		// Send email via Resend
		client := resend.NewClient(os.Getenv("RESEND_API_KEY"))
		_, sendErr := client.Emails.Send(&resend.SendEmailRequest{
			From:    "onboarding@resend.dev",
			To:      []string{body.Email},
			Subject: "CapyPlan Password Reset Request",
			Html: `<p>A password reset request was submitted for this email address.</p>
<p>Your temporary password is:</p>
<pre>` + code + `</pre>
<p>Please login and change your password immediately in your settings menu.</p>
<p>Best regards,</p>
<p>The CapyPlan Team</p>`,
		})
		if sendErr != nil {
			// rollback password
			db.DB.Exec(`UPDATE users SET password = $1 WHERE email = $2`, originalPassword, body.Email)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]bool{"success": false})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	return r
}

// getUserIDFromToken is a helper to extract user ID from JWT — reusable across routes
func getUserIDFromToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		return "", false
	}
	userID, err := extractUserIDFromToken(parts[1])
	if err != nil {
		return "", false
	}
	return userID, true
}
