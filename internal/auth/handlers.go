package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Handlers struct {
	DB  *sql.DB
	JWT *JWT
}

type creds struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

func NewHandlers(db *sql.DB, jwt *JWT) *Handlers {
	return &Handlers{DB: db, JWT: jwt}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var c creds
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := validateRegistration(c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	_, err = h.DB.Exec(
		`INSERT INTO users (username, password, email, phone) VALUES ($1, $2, $3, $4)`,
		c.Username,
		string(hash),
		strings.ToLower(c.Email),
		c.Phone,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "users_username_key") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "username already exists"})
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "users_email_key") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "user created successfully"})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var c creds
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if strings.TrimSpace(c.Username) == "" || c.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}

	var userID int64
	var username string
	var hash string
	err := h.DB.QueryRow(
		`SELECT id, username, password FROM users WHERE username = $1`,
		c.Username,
	).Scan(&userID, &username, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "wrong credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "wrong credentials"})
		return
	}

	token, err := h.JWT.Sign(userID, username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func validateRegistration(c creds) error {
	username := strings.TrimSpace(c.Username)
	if len(username) < 3 || len(username) > 32 {
		return errors.New("invalid username")
	}
	if len(c.Password) < 6 {
		return errors.New("invalid password")
	}
	if !strings.Contains(c.Email, "@") || strings.TrimSpace(c.Email) == "" {
		return errors.New("invalid email")
	}
	return nil
}
