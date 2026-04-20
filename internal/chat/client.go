package chat

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second // max time to write to a client
	pongWait   = 60 * time.Second // how long we wait for a pong
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 4 * 1024 // max message size accepted
)

type Client struct {
	conn        *websocket.Conn
	send        chan []byte
	userID      int64
	username    string
	joinedRooms map[int64]struct{}
}

type inboundEvent struct {
	Type     string `json:"type"`
	RoomID   int64  `json:"room_id"`
	RoomName string `json:"room_name"`
	Content  string `json:"content"`
}

func (c *Client) readPump(onEvent func(inboundEvent)) {
	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var in inboundEvent
		if err := json.Unmarshal(data, &in); err != nil {
			continue
		}
		onEvent(in)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
