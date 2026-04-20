package chat

import (
	"database/sql"
	"errors"
	"strings"

	"realtime_chat/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListRooms() ([]models.Room, error) {
	rows, err := r.db.Query(`SELECT id, name FROM rooms ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]models.Room, 0)
	for rows.Next() {
		var room models.Room
		if err := rows.Scan(&room.ID, &room.Name); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (r *Repository) CreateRoom(name string) (models.Room, error) {
	name = strings.TrimSpace(name)
	var room models.Room
	err := r.db.QueryRow(`INSERT INTO rooms(name) VALUES ($1) RETURNING id, name`, name).Scan(&room.ID, &room.Name)
	return room, err
}

func (r *Repository) RoomExists(roomID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)`, roomID).Scan(&exists)
	return exists, err
}

func (r *Repository) FindRoomIDByName(name string) (int64, error) {
	var roomID int64
	err := r.db.QueryRow(`SELECT id FROM rooms WHERE name = $1`, strings.TrimSpace(name)).Scan(&roomID)
	return roomID, err
}

func (r *Repository) SaveMessage(roomID, userID int64, content string) (models.Message, error) {
	var message models.Message
	err := r.db.QueryRow(
		`INSERT INTO messages(room_id, user_id, content) VALUES ($1, $2, $3)
		 RETURNING id, room_id, user_id, content, created_at`,
		roomID, userID, strings.TrimSpace(content),
	).Scan(&message.ID, &message.RoomID, &message.UserID, &message.Content, &message.CreatedAt)
	return message, err
}

func (r *Repository) ListMessages(roomID int64) ([]models.Message, error) {
	rows, err := r.db.Query(
		`SELECT m.id, m.room_id, m.content, m.created_at, u.username
		 FROM messages m
		 JOIN users u ON m.user_id = u.id
		 WHERE m.room_id = $1
		 ORDER BY m.created_at ASC`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]models.Message, 0)
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.Content, &msg.CreatedAt, &msg.Username); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

var ErrRoomExists = errors.New("room already exists")
