package chat

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"realtime_chat/internal/auth"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub  *Hub
	repo *Repository
	jwt  *auth.JWT
}

func NewWSHandler(hub *Hub, repo *Repository, jwt *auth.JWT) *WSHandler {
	return &WSHandler{hub: hub, repo: repo, jwt: jwt}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	claims, err := h.jwt.Verify(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		conn:        conn,
		send:        make(chan []byte, 256),
		userID:      claims.UserID,
		username:    claims.Username,
		joinedRooms: make(map[int64]struct{}),
	}

	h.hub.Register(client)

	go client.writePump()
	go func() {
		defer func() {
			h.hub.Unregister(client)
			_ = conn.Close()
		}()

		client.readPump(func(evt inboundEvent) {
			roomID, err := h.resolveRoomID(evt.RoomID, evt.RoomName)
			if evt.Type == "join_room" || evt.Type == "leave_room" || evt.Type == "send_message" {
				if err != nil || roomID <= 0 {
					sendClientError(client, "room not found")
					return
				}
			}

			switch evt.Type {
			case "join_room":
				h.hub.Join(client, roomID)
			case "leave_room":
				h.hub.Leave(client, roomID)
			case "send_message":
				if strings.TrimSpace(evt.Content) == "" {
					sendClientError(client, "invalid message payload")
					return
				}
				saved, err := h.repo.SaveMessage(roomID, client.userID, evt.Content)
				if err != nil {
					sendClientError(client, "failed to persist message")
					return
				}
				payload, _ := json.Marshal(outboundEvent{
					Type:      "message",
					RoomID:    roomID,
					Username:  client.username,
					Content:   saved.Content,
					CreatedAt: saved.CreatedAt.UTC().Format(time.RFC3339),
				})
				h.hub.Broadcast(roomID, payload)
			default:
				sendClientError(client, "unknown event")
			}
		})
	}()
}

func (h *WSHandler) resolveRoomID(roomID int64, roomName string) (int64, error) {
	if roomID > 0 {
		exists, err := h.repo.RoomExists(roomID)
		if err != nil || !exists {
			return 0, sql.ErrNoRows
		}
		return roomID, nil
	}
	if strings.TrimSpace(roomName) == "" {
		return 0, sql.ErrNoRows
	}
	return h.repo.FindRoomIDByName(roomName)
}

func sendClientError(c *Client, message string) {
	payload, _ := json.Marshal(outboundEvent{Type: "error", Error: message})
	select {
	case c.send <- payload:
	default:
	}
}
