package chat

type outboundEvent struct {
	Type      string `json:"type"`
	RoomID    int64  `json:"room_id"`
	Username  string `json:"username,omitempty"`
	Content   string `json:"content,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Error     string `json:"error,omitempty"`
}
