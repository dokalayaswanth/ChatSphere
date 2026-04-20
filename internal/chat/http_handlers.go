package chat

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"realtime_chat/internal/auth"
)

type HTTPHandlers struct {
	repo *Repository
	jwt  *auth.JWT
}

func NewHTTPHandlers(repo *Repository, jwt *auth.JWT) *HTTPHandlers {
	return &HTTPHandlers{repo: repo, jwt: jwt}
}

func (h *HTTPHandlers) Rooms(w http.ResponseWriter, r *http.Request) {
	_, ok := h.requireAuth(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		rooms, err := h.repo.ListRooms()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
		writeJSON(w, http.StatusOK, rooms)
	case http.MethodPost:
		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if strings.TrimSpace(payload.Name) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room name is required"})
			return
		}
		room, err := h.repo.CreateRoom(payload.Name)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "rooms_name_key") {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "room already exists"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
		writeJSON(w, http.StatusCreated, room)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *HTTPHandlers) Messages(w http.ResponseWriter, r *http.Request) {
	_, ok := h.requireAuth(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	roomIDStr := r.URL.Query().Get("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil || roomID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid room_id"})
		return
	}

	messages, err := h.repo.ListMessages(roomID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (h *HTTPHandlers) requireAuth(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return auth.Claims{}, false
	}
	claims, err := h.jwt.Verify(token)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return auth.Claims{}, false
	}
	return claims, true
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
